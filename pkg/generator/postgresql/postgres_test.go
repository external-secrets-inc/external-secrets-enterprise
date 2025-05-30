package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"testing"

	"github.com/docker/go-connections/nat"

	"github.com/jackc/pgx/v5"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

const (
	testUser                = "generated_user"
	testPass                = "strongpassword"
	testNamespace           = "default"
	testSecretName          = "testpass"
	testSecretKey           = "password"
	testGeneratedSecretName = "userpass"
)

type generatorMockClient struct {
	client.Client
	userPassword []byte
	t            *testing.T
}

func (m generatorMockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	m.t.Helper()
	if key.Name == testSecretName {
		obj.(*corev1.Secret).Data = map[string][]byte{
			testSecretKey: []byte(testPass),
		}
	} else if key.Name == testGeneratedSecretName {
		obj.(*corev1.Secret).Data = map[string][]byte{
			testSecretKey: m.userPassword,
		}
	}
	return nil
}

type PostgresTestSuite struct {
	suite.Suite
	ctx    context.Context
	client generatorMockClient
	db     *pgx.Conn
	host   string
	port   nat.Port
	pg     *tcpostgres.PostgresContainer
}

func TestPostgresGeneratorTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresTestSuite))
}

func (s *PostgresTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.client = generatorMockClient{t: s.T()}

	s.host = "localhost"
	pgContainer, err := tcpostgres.Run(s.ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("postgres"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword(testPass),
		// testcontainers.WithExposedPorts("5432/tcp"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(s.T(), err)
	s.pg = pgContainer
	s.port, err = pgContainer.MappedPort(s.ctx, "5432/tcp")
	require.NoError(s.T(), err)
	s.T().Logf("Port: %s", string(s.port))

	connStr, err := pgContainer.ConnectionString(s.ctx, "sslmode=disable")
	s.T().Logf("Postgres connection string: %s", connStr)
	require.NoError(s.T(), err)

	conn, err := pgx.Connect(s.ctx, connStr)
	require.NoError(s.T(), err)
	require.NoError(s.T(), conn.Ping(s.ctx))

	s.db = conn

	s.T().Cleanup(func() {
		conn.Close(s.ctx)
		if err := testcontainers.TerminateContainer(pgContainer); err != nil {
			s.T().Logf("failed to terminate container: %s", err)
		}
	})
}

func newGeneratorSpec(t *testing.T, host, port, username string, destructive bool) *genv1alpha1.PostgreSql {
	t.Helper()

	return &genv1alpha1.PostgreSql{
		Spec: genv1alpha1.PostgreSqlSpec{
			Host:     host,
			Port:     port,
			Database: "postgres",
			Auth: genv1alpha1.PostgreSqlAuth{
				Username: "postgres",
				Password: genv1alpha1.SecretKeySelector{
					Name: testSecretName,
					Key:  testSecretKey,
				},
			},
			User: &genv1alpha1.PostgreSqlUser{
				Username:           username,
				Attributes:         []string{"CREATEDB"},
				Roles:              []string{"pg_read_all_data", "customrole"},
				DestructiveCleanup: destructive,
			},
		},
	}
}

func (s *PostgresTestSuite) TestGenerateAndCleanupUser() {
	user := fmt.Sprintf("%s_TestGenerate", testUser)

	spec := newGeneratorSpec(s.T(), "localhost", s.port.Port(), user, true)

	specJSON, err := yaml.Marshal(spec)
	require.NoError(s.T(), err)

	gen := &Generator{}

	// Call Generate
	result, statusRaw, err := gen.Generate(s.ctx, &apiextensions.JSON{Raw: specJSON}, s.client, testNamespace)
	require.NoError(s.T(), err)
	require.Contains(s.T(), result, "user")
	require.Contains(s.T(), result, "password")
	regex := regexp.MustCompile(fmt.Sprintf(`%s_[a-zA-Z0-9]{8}`, user))
	assert.Regexp(s.T(), regex, string(result["user"]))

	generatedUser := string(result["user"])
	// Verify attributes
	var (
		rolcanlogin    bool
		rolcreatedb    bool
		rolcreaterole  bool
		rolsuper       bool
		rolreplication bool
	)
	row := s.db.QueryRow(s.ctx, `
		SELECT rolcanlogin, rolcreatedb, rolcreaterole, rolsuper, rolreplication
		FROM pg_roles
		WHERE rolname = $1
	`, generatedUser)
	err = row.Scan(&rolcanlogin, &rolcreatedb, &rolcreaterole, &rolsuper, &rolreplication)
	require.NoError(s.T(), err)

	assert.True(s.T(), rolcanlogin, "user should be able to login")
	assert.True(s.T(), rolcreatedb, "user should have CREATEDB")
	assert.False(s.T(), rolcreaterole, "user should not have CREATEROLE")
	assert.False(s.T(), rolsuper, "user should not be SUPERUSER")
	assert.False(s.T(), rolreplication, "user should not have REPLICATION")

	// Verify granted roles
	rows, err := s.db.Query(s.ctx, `
		SELECT r.rolname
		FROM pg_auth_members m
		JOIN pg_roles r ON r.oid = m.roleid
		JOIN pg_roles u ON u.oid = m.member
		WHERE u.rolname = $1
	`, generatedUser)
	require.NoError(s.T(), err)

	defer rows.Close()
	var grantedRoles []string
	for rows.Next() {
		var role string
		require.NoError(s.T(), rows.Scan(&role))
		grantedRoles = append(grantedRoles, role)
	}

	require.NoError(s.T(), rows.Err())

	assert.Contains(s.T(), grantedRoles, "pg_read_all_data")
	assert.Contains(s.T(), grantedRoles, "customrole")

	// Cleanup
	err = gen.Cleanup(s.ctx, &apiextensions.JSON{Raw: specJSON}, statusRaw, s.client, testNamespace)
	require.NoError(s.T(), err)

	// Verify user was dropped
	row = s.db.QueryRow(s.ctx, `SELECT 1 FROM pg_roles WHERE rolname = $1`, generatedUser)
	var dummy int
	err = row.Scan(&dummy)
	assert.ErrorIs(s.T(), err, sql.ErrNoRows)
}

func (s *PostgresTestSuite) TestNonDestructiveCleanup() {
	user := fmt.Sprintf("%s_NonDestructive", testUser)

	spec := newGeneratorSpec(s.T(), "localhost", s.port.Port(), user, false)
	spec.Spec.User.Attributes = []string{"SUPERUSER"}
	specJSON, err := yaml.Marshal(spec)
	require.NoError(s.T(), err)

	gen := &Generator{}
	result, rawStatus, err := gen.Generate(s.ctx, &apiextensions.JSON{Raw: specJSON}, s.client, testNamespace)
	require.NoError(s.T(), err)
	require.Contains(s.T(), result, "user")
	require.Contains(s.T(), result, "password")

	generatedUser := string(result["user"])
	password := string(result["password"])

	userSpec := newGeneratorSpec(s.T(), "localhost", s.port.Port(), generatedUser, false)
	userSpec.Spec.Auth = genv1alpha1.PostgreSqlAuth{
		Username: generatedUser,
		Password: genv1alpha1.SecretKeySelector{
			Name: testGeneratedSecretName,
			Key:  testSecretKey,
		},
	}
	userClient := generatorMockClient{t: s.T(), userPassword: []byte(password)}
	userDB, err := newConnection(s.ctx, &userSpec.Spec, userClient, testNamespace)
	require.NoError(s.T(), err)
	defer userDB.Close(s.ctx)

	_, err = userDB.Exec(s.ctx, `CREATE TABLE cleanup_test (id INT)`)
	require.NoError(s.T(), err)

	err = gen.Cleanup(s.ctx, &apiextensions.JSON{Raw: specJSON}, rawStatus, s.client, testNamespace)
	require.NoError(s.T(), err)

	row := s.db.QueryRow(s.ctx, `SELECT 1 FROM pg_roles WHERE rolname = $1`, generatedUser)
	var dummy int
	err = row.Scan(&dummy)
	assert.ErrorIs(s.T(), err, sql.ErrNoRows)

	row = s.db.QueryRow(s.ctx, `
		SELECT tableowner
		FROM pg_tables
		WHERE tablename = 'cleanup_test'
	`)
	var owner string
	err = row.Scan(&owner)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "postgres", owner)

	_, err = s.db.Exec(s.ctx, `DROP TABLE IF EXISTS cleanup_test`)
	require.NoError(s.T(), err)
}

func (s *PostgresTestSuite) TestDestructiveCleanup() {
	user := fmt.Sprintf("%s_Destructive", testUser)

	spec := newGeneratorSpec(s.T(), "localhost", s.port.Port(), user, true)
	spec.Spec.User.Attributes = []string{"SUPERUSER"}
	specJSON, err := yaml.Marshal(spec)
	require.NoError(s.T(), err)

	gen := &Generator{}
	result, rawStatus, err := gen.Generate(s.ctx, &apiextensions.JSON{Raw: specJSON}, s.client, testNamespace)
	require.NoError(s.T(), err)
	require.Contains(s.T(), result, "user")
	require.Contains(s.T(), result, "password")

	generatedUser := string(result["user"])
	password := string(result["password"])

	userSpec := newGeneratorSpec(s.T(), "localhost", s.port.Port(), generatedUser, true)
	userSpec.Spec.Auth = genv1alpha1.PostgreSqlAuth{
		Username: generatedUser,
		Password: genv1alpha1.SecretKeySelector{
			Name: testGeneratedSecretName,
			Key:  testSecretKey,
		},
	}
	userClient := generatorMockClient{t: s.T(), userPassword: []byte(password)}
	userDB, err := newConnection(s.ctx, &userSpec.Spec, userClient, testNamespace)
	require.NoError(s.T(), err)
	defer userDB.Close(s.ctx)

	_, err = userDB.Exec(s.ctx, `CREATE TABLE cleanup_test (id INT)`)
	require.NoError(s.T(), err)

	err = gen.Cleanup(s.ctx, &apiextensions.JSON{Raw: specJSON}, rawStatus, s.client, testNamespace)
	require.NoError(s.T(), err)

	row := s.db.QueryRow(s.ctx, `SELECT 1 FROM pg_roles WHERE rolname = $1`, generatedUser)
	var dummy int
	err = row.Scan(&dummy)
	assert.ErrorIs(s.T(), err, sql.ErrNoRows)

	row = s.db.QueryRow(s.ctx, `
		SELECT 1
		FROM pg_tables
		WHERE tablename = 'cleanup_test'
	`)
	err = row.Scan(&dummy)
	assert.ErrorIs(s.T(), err, sql.ErrNoRows)
}

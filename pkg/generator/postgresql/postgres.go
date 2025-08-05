// Copyright External Secrets Inc. All Rights Reserved

package postgresql

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"slices"
	"strings"

	"time"

	"github.com/jackc/pgx/v5"
	cronV3 "github.com/robfig/cron/v3"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtimeyaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/generator/password"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

type Generator struct{}

const (
	defaultPort       = "5432"
	defaultUser       = "postgres"
	defaultDbName     = "postgres"
	defaultSuffixSize = 8
)

var mapAttributes = map[string]genv1alpha1.PostgreSqlUserAttributesEnum{
	string(genv1alpha1.PostgreSqlUserSuperUser):   genv1alpha1.PostgreSqlUserSuperUser,
	string(genv1alpha1.PostgreSqlUserCreateDb):    genv1alpha1.PostgreSqlUserCreateDb,
	string(genv1alpha1.PostgreSqlUserCreateRole):  genv1alpha1.PostgreSqlUserCreateRole,
	string(genv1alpha1.PostgreSqlUserReplication): genv1alpha1.PostgreSqlUserReplication,
	string(genv1alpha1.PostgreSqlUserNoInherit):   genv1alpha1.PostgreSqlUserNoInherit,
	string(genv1alpha1.PostgreSqlUserByPassRls):   genv1alpha1.PostgreSqlUserByPassRls,
	"CONNECTION_LIMIT":                            genv1alpha1.PostgreSqlUserConnectionLimit,
	string(genv1alpha1.PostgreSqlUserLogin):       genv1alpha1.PostgreSqlUserLogin,
	string(genv1alpha1.PostgreSqlUserPassword):    genv1alpha1.PostgreSqlUserPassword,
}

func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, err
	}

	db, err := newConnection(ctx, &res.Spec, kube, namespace)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create db connection: %w", err)
	}
	defer func() {
		err := db.Close(ctx)
		if err != nil {
			fmt.Printf("failed to close db: %v", err)
		}
	}()

	err = db.Ping(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to ping the database: %w", err)
	}

	cleanupPolicy, err := g.GetCleanupPolicy(jsonSpec)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get cleanup policy: %w", err)
	}
	if cleanupPolicy != nil && cleanupPolicy.Type == genv1alpha1.IdleCleanupPolicy {
		err = setupObservation(ctx, db)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to setup observation: %w", err)
		}
		manifest, err := getCronjobManifest(*res)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to get cronjob manifest: %w", err)
		}
		err = applyCronJob(ctx, kube, manifest, metav1.OwnerReference{
			UID:        res.UID,
			APIVersion: res.APIVersion,
			Kind:       res.Kind,
			Name:       res.Name,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("unable to apply cronjob: %w", err)
		}
	}

	user, err := createUser(ctx, db, &res.Spec)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create or update user: %w", err)
	}

	username, ok := user["username"]
	if !ok {
		return nil, nil, fmt.Errorf("user not found in response")
	}

	rawState, err := json.Marshal(&genv1alpha1.PostgreSqlUserState{
		Username: string(username),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to marshal state: %w", err)
	}

	return user, &apiextensions.JSON{Raw: rawState}, nil
}

func (g *Generator) Cleanup(ctx context.Context, jsonSpec *apiextensions.JSON, previousStatus genv1alpha1.GeneratorProviderState, kclient client.Client, namespace string) error {
	if previousStatus == nil {
		return fmt.Errorf("missing previous status")
	}
	status, err := parseStatus(previousStatus.Raw)
	if err != nil {
		return err
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return err
	}
	db, err := newConnection(ctx, &res.Spec, kclient, namespace)
	if err != nil {
		return err
	}
	defer func() {
		err := db.Close(ctx)
		if err != nil {
			fmt.Printf("failed to close db: %v", err)
		}
	}()

	err = db.Ping(ctx)
	if err != nil {
		return fmt.Errorf("unable to ping the database: %w", err)
	}

	err = dropUser(ctx, db, status.Username, res.Spec)
	if err != nil {
		return fmt.Errorf("unable to drop user: %w", err)
	}

	return nil
}

func (g *Generator) GetCleanupPolicy(obj *apiextensions.JSON) (*genv1alpha1.CleanupPolicy, error) {
	res, err := parseSpec(obj.Raw)
	if err != nil {
		return nil, err
	}
	if res.Spec.CleanupPolicy == nil {
		return nil, nil
	}

	policy := genv1alpha1.CleanupPolicy{
		Type:        res.Spec.CleanupPolicy.Type,
		IdleTimeout: res.Spec.CleanupPolicy.IdleTimeout,
		GracePeriod: res.Spec.CleanupPolicy.GracePeriod,
	}
	return &policy, nil
}

func (g *Generator) LastActivityTime(ctx context.Context, obj *apiextensions.JSON, state genv1alpha1.GeneratorProviderState, kube client.Client, namespace string) (time.Time, bool, error) {
	status, err := parseStatus(state.Raw)
	if err != nil {
		return time.Time{}, false, err
	}
	res, err := parseSpec(obj.Raw)
	if err != nil {
		return time.Time{}, false, err
	}
	db, err := newConnection(ctx, &res.Spec, kube, namespace)
	if err != nil {
		return time.Time{}, false, err
	}
	defer func() {
		err := db.Close(ctx)
		if err != nil {
			fmt.Printf("failed to close db: %v", err)
		}
	}()

	err = db.Ping(ctx)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("unable to ping the database: %w", err)
	}

	lastActivity, err := getUserActivity(ctx, db, status.Username)
	if err != nil {
		return time.Time{}, false, err
	}
	return lastActivity, true, nil
}

func (g *Generator) GetKeys() map[string]string {
	return map[string]string{
		"username": "PostgreSQL database username",
		"password": "PostgreSQL user password",
	}
}

func newConnection(ctx context.Context, spec *genv1alpha1.PostgreSqlSpec, kclient client.Client, ns string) (*pgx.Conn, error) {
	dbName := defaultDbName
	if spec.Database != "" {
		dbName = spec.Database
	}

	port := defaultPort
	if spec.Port != "" {
		port = spec.Port
	}

	username := defaultUser
	if spec.Auth.Username != "" {
		username = spec.Auth.Username
	}
	password, err := resolvers.SecretKeyRef(ctx, kclient, resolvers.EmptyStoreKind, ns, &esmeta.SecretKeySelector{
		Namespace: &ns,
		Name:      spec.Auth.Password.Name,
		Key:       spec.Auth.Password.Key,
	})
	if err != nil {
		return nil, err
	}

	psqlInfo := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		spec.Host, port, username, password, dbName,
	)

	return pgx.Connect(ctx, psqlInfo)
}

func createSessionObservationTable(ctx context.Context, db *pgx.Conn) error {
	query := `
CREATE TABLE IF NOT EXISTS session_observation (
    pid              INTEGER     PRIMARY KEY,
    usename          TEXT        NOT NULL,
    client_addr      INET,
    application_name TEXT,
    state            TEXT,
    state_change     TIMESTAMPTZ,
    first_seen       TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen        TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

	if _, err := db.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to create session_observation table: %w", err)
	}

	return nil
}

func createSessionSnapshotFunction(ctx context.Context, db *pgx.Conn) error {
	query := `
CREATE OR REPLACE FUNCTION snapshot_pg_stat_activity() RETURNS void AS $$
BEGIN
  INSERT INTO session_observation (pid, usename, client_addr, application_name, state, state_change)
  SELECT
    pid,
    usename,
    client_addr,
    application_name,
    state,
    state_change
  FROM pg_stat_activity
  WHERE pid <> pg_backend_pid()  -- ignore yourself
  AND usename IS NOT NULL
  ON CONFLICT (pid)
  DO UPDATE SET
    state = EXCLUDED.state,
    state_change = EXCLUDED.state_change,
    last_seen = now();
END;
$$ LANGUAGE plpgsql;
`

	if _, err := db.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to create session_observation table: %w", err)
	}

	return nil
}

//go:embed cronjob-template.yaml
var cronjobTemplate []byte

func getCronjobManifest(genSpec genv1alpha1.PostgreSql) (string, error) {
	tplContent := string(cronjobTemplate)
	cronExpression := "* * * * *"
	if genSpec.Spec.CleanupPolicy.ActivityTrackingCron != nil {
		cronExpression = *genSpec.Spec.CleanupPolicy.ActivityTrackingCron
	}
	_, err := cronV3.ParseStandard(cronExpression)
	if err != nil {
		return "", fmt.Errorf("failed to parse cron expression: %w", err)
	}

	params := map[string]string{
		"Name":       sanitizeName(fmt.Sprintf("psql-%s-%s-session-observation-cronjob", genSpec.Spec.Host, genSpec.Spec.Port)),
		"Namespace":  genSpec.GetNamespace(),
		"Schedule":   cronExpression,
		"SecretName": genSpec.Spec.Auth.Password.Name,
		"SecretKey":  genSpec.Spec.Auth.Password.Key,
		"PgUser":     genSpec.Spec.Auth.Username,
		"Host":       genSpec.Spec.Host,
		"Port":       genSpec.Spec.Port,
		"Database":   genSpec.Spec.Database,
	}

	tpl, err := template.New("cronjob").Parse(tplContent)
	if err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}
	var rendered bytes.Buffer
	if err := tpl.Execute(&rendered, params); err != nil {
		return "", fmt.Errorf("template execution error: %w", err)
	}

	return rendered.String(), nil
}

func sanitizeName(name string) string {
	name = strings.ToLower(name)

	// replace invalid chars
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	sanitized := b.String()

	sanitized = strings.Trim(sanitized, "-")

	if len(sanitized) > 63 {
		sanitized = sanitized[:63]
	}

	// trim again in case cut created trailing dash
	sanitized = strings.Trim(sanitized, "-")

	return sanitized
}

var applyCronJob = func(ctx context.Context, c client.Client, manifest string, ownerRef metav1.OwnerReference) error {
	dec := runtimeyaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	obj := &unstructured.Unstructured{}

	_, gvk, err := dec.Decode([]byte(manifest), nil, obj)
	if err != nil {
		return fmt.Errorf("failed to decode manifest: %w", err)
	}
	ns := obj.GetNamespace()
	obj.SetGroupVersionKind(*gvk)

	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(obj.GroupVersionKind())
	err = c.Get(ctx, client.ObjectKey{Namespace: ns, Name: obj.GetName()}, existing)
	isNew := false

	var owners []metav1.OwnerReference
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to get existing resource: %w", err)
		}
		owners = []metav1.OwnerReference{ownerRef}
		isNew = true
	} else {
		owners = existing.GetOwnerReferences()
		found := false
		for _, or := range owners {
			if or.UID == ownerRef.UID {
				found = true
				break
			}
		}
		if !found {
			owners = append(owners, ownerRef)
		}
		obj.SetResourceVersion(existing.GetResourceVersion())
	}
	obj.SetOwnerReferences(owners)

	if isNew {
		if err := c.Create(ctx, obj); err != nil {
			return fmt.Errorf("failed to create CronJob: %w", err)
		}
	} else {
		if err := c.Update(ctx, obj); err != nil {
			return fmt.Errorf("failed to update CronJob: %w", err)
		}
	}
	return nil
}

func getUserActivity(ctx context.Context, db *pgx.Conn, username string) (time.Time, error) {
	var lastSeen time.Time

	const sqlQuery = `
        SELECT last_seen
        FROM session_observation
        WHERE usename = $1
        ORDER BY last_seen DESC
        LIMIT 1;
    `

	err := db.QueryRow(ctx, sqlQuery, username).Scan(&lastSeen)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Unix(0, 0), nil
		}
		return time.Time{}, fmt.Errorf("failed to get user activity: %w", err)
	}

	return lastSeen, nil
}

func setupObservation(ctx context.Context, db *pgx.Conn) error {
	if err := createSessionObservationTable(ctx, db); err != nil {
		return fmt.Errorf("failed to create session_observation table: %w", err)
	}
	if err := createSessionSnapshotFunction(ctx, db); err != nil {
		return fmt.Errorf("failed to create session_observation function: %w", err)
	}
	return nil
}

func getExistingRoles(ctx context.Context, db *pgx.Conn) ([]string, error) {
	var current_rows = make([]string, 0)
	rows, err := db.Query(ctx, "SELECT rolname FROM pg_roles")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var rolname string
		err = rows.Scan(&rolname)
		if err != nil {
			return nil, err
		}
		current_rows = append(current_rows, rolname)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return current_rows, nil
}

func addRolesAttributesToQueryString(query *strings.Builder, attributes []genv1alpha1.PostgreSqlUserAttribute) {
	if len(attributes) > 0 {
		query.WriteString(" WITH ")
		for i, attr := range attributes {
			if i > 0 {
				query.WriteString(" ")
			}
			if attr.Value != nil {
				if string(mapAttributes[attr.Name]) == string(genv1alpha1.PostgreSqlUserPassword) {
					fmt.Fprintf(query, `%s '%s'`, string(mapAttributes[attr.Name]), *attr.Value)
				} else {
					fmt.Fprintf(query, `%s %s`, string(mapAttributes[attr.Name]), *attr.Value)
				}
			} else {
				query.WriteString(string(mapAttributes[attr.Name]))
			}
		}
	}
}

func createRole(ctx context.Context, db *pgx.Conn, roleName string, attributes []genv1alpha1.PostgreSqlUserAttribute) error {
	var query strings.Builder
	query.WriteString(fmt.Sprintf("CREATE ROLE %s", pgx.Identifier{roleName}.Sanitize()))
	addRolesAttributesToQueryString(&query, attributes)
	_, err := db.Exec(ctx, query.String())
	return err
}

func updateRole(ctx context.Context, db *pgx.Conn, roleName string, attributes []genv1alpha1.PostgreSqlUserAttribute) error {
	var query strings.Builder
	query.WriteString(fmt.Sprintf("ALTER ROLE %s", pgx.Identifier{roleName}.Sanitize()))
	addRolesAttributesToQueryString(&query, attributes)
	_, err := db.Exec(ctx, query.String())
	return err
}

func resetRole(ctx context.Context, db *pgx.Conn, roleName string) error {
	sanitizedRole := pgx.Identifier{roleName}.Sanitize()

	_, err := db.Exec(ctx, fmt.Sprintf(`
		ALTER ROLE %s WITH NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOLOGIN NOREPLICATION NOBYPASSRLS
	`, sanitizedRole))
	if err != nil {
		return fmt.Errorf("failed to reset attributes for role %s: %w", roleName, err)
	}

	rows, err := db.Query(ctx, `
		SELECT r.rolname
		FROM pg_auth_members m
		JOIN pg_roles r ON r.oid = m.roleid
		JOIN pg_roles u ON u.oid = m.member
		WHERE u.rolname = $1
	`, roleName)
	if err != nil {
		return fmt.Errorf("failed to list granted roles for %s: %w", roleName, err)
	}
	defer rows.Close()

	var grantedRoles []string
	for rows.Next() {
		var grantedRole string
		if err := rows.Scan(&grantedRole); err != nil {
			return fmt.Errorf("failed to scan granted role: %w", err)
		}
		grantedRoles = append(grantedRoles, pgx.Identifier{grantedRole}.Sanitize())
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating granted roles: %w", err)
	}

	rolesCSV := strings.Join(grantedRoles, ", ")

	_, err = db.Exec(ctx, fmt.Sprintf("REVOKE %s FROM %s", rolesCSV, sanitizedRole))
	if err != nil {
		return fmt.Errorf("failed to revoke roles [%s] from %s: %w", rolesCSV, roleName, err)
	}

	return nil
}

func createUser(ctx context.Context, db *pgx.Conn, spec *genv1alpha1.PostgreSqlSpec) (map[string][]byte, error) {
	username := spec.User.Username
	suffixSize := defaultSuffixSize
	if spec.User.SuffixSize != nil {
		suffixSize = *spec.User.SuffixSize
	}
	suffix, err := utils.GenerateRandomString(suffixSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random suffix: %w", err)
	}

	if suffix != "" {
		username = fmt.Sprintf("%s_%s", username, suffix)
	}

	current_roles, err := getExistingRoles(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing roles: %w", err)
	}

	pass, err := generatePassword(genv1alpha1.Password{
		Spec: genv1alpha1.PasswordSpec{
			SymbolCharacters: ptr.To("~!@#$%^&*()_+-={}|[]:<>?,./"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	spec.User.Attributes = append(spec.User.Attributes,
		genv1alpha1.PostgreSqlUserAttribute{
			Name: string(genv1alpha1.PostgreSqlUserLogin),
		}, genv1alpha1.PostgreSqlUserAttribute{
			Name:  string(genv1alpha1.PostgreSqlUserPassword),
			Value: ptr.To(string(pass)),
		},
	)

	if !slices.Contains(current_roles, username) {
		err = createRole(ctx, db, username, spec.User.Attributes)
		if err != nil {
			return nil, fmt.Errorf("failed to create role %s: %w", username, err)
		}
	} else {
		err = resetRole(ctx, db, username)
		if err != nil {
			return nil, fmt.Errorf("failed to reset role %s: %w", username, err)
		}
		err = updateRole(ctx, db, username, spec.User.Attributes)
		if err != nil {
			return nil, fmt.Errorf("failed to create role %s: %w", username, err)
		}
	}

	err = grantRolesToUser(ctx, db, username, spec.User.Roles, current_roles)
	if err != nil {
		return nil, fmt.Errorf("failed to add roles to user %s: %w", username, err)
	}

	return map[string][]byte{
		"username": []byte(username),
		"password": pass,
	}, nil
}

func grantRolesToUser(ctx context.Context, db *pgx.Conn, username string, roles, current_roles []string) error {
	sanitizedUsername := pgx.Identifier{username}.Sanitize()

	toGrant := make([]string, 0, len(roles))
	for _, role := range roles {
		if !slices.Contains(current_roles, role) {
			if err := createRole(ctx, db, role, nil); err != nil {
				return fmt.Errorf("failed to create role %s: %w", role, err)
			}
		}
		toGrant = append(toGrant, pgx.Identifier{role}.Sanitize())
	}

	if len(toGrant) == 0 {
		return nil
	}

	rolesCSV := strings.Join(toGrant, ", ")
	query := fmt.Sprintf("GRANT %s TO %s", rolesCSV, sanitizedUsername)

	if _, err := db.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to grant roles [%s] to user %s: %w", rolesCSV, username, err)
	}

	return nil
}

func dropUser(ctx context.Context, db *pgx.Conn, username string, spec genv1alpha1.PostgreSqlSpec) error {
	sanitizedUsername := pgx.Identifier{username}.Sanitize()
	if !spec.User.DestructiveCleanup {
		reassignToUser := spec.Auth.Username
		if spec.User.ReassignTo != nil && *spec.User.ReassignTo != "" {
			reassignToUser = *spec.User.ReassignTo
		}

		current_roles, err := getExistingRoles(ctx, db)
		if err != nil {
			return fmt.Errorf("failed to get existing roles: %w", err)
		}
		if !slices.Contains(current_roles, reassignToUser) {
			err = createRole(ctx, db, reassignToUser, nil)
			if err != nil {
				return fmt.Errorf("failed to create role %s: %w", reassignToUser, err)
			}
		}

		_, err = db.Exec(ctx, fmt.Sprintf(`REASSIGN OWNED BY %s TO %s`, sanitizedUsername, pgx.Identifier{reassignToUser}.Sanitize()))
		if err != nil {
			return fmt.Errorf("failed to reassign owned by %s to %s: %w", username, reassignToUser, err)
		}
	}
	dropQueries := []string{
		`DROP OWNED BY %s`,
		`DROP ROLE %s`,
	}
	for _, query := range dropQueries {
		_, err := db.Exec(ctx, fmt.Sprintf(query, sanitizedUsername))
		if err != nil {
			return err
		}
	}
	return nil
}

func generatePassword(
	passSpec genv1alpha1.Password,
) ([]byte, error) {
	gen := password.Generator{}
	rawPassSpec, err := yaml.Marshal(passSpec)
	if err != nil {
		return nil, err
	}
	passMap, _, err := gen.Generate(context.TODO(), &apiextensions.JSON{Raw: rawPassSpec}, nil, "")

	if err != nil {
		return nil, err
	}

	pass, ok := passMap["password"]
	if !ok {
		return nil, fmt.Errorf("password not found in generated map")
	}
	return pass, nil
}

func parseSpec(data []byte) (*genv1alpha1.PostgreSql, error) {
	var spec genv1alpha1.PostgreSql
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func parseStatus(data []byte) (*genv1alpha1.PostgreSqlUserState, error) {
	var state genv1alpha1.PostgreSqlUserState
	err := json.Unmarshal(data, &state)
	if err != nil {
		return nil, err
	}
	return &state, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.PostgreSqlKind, &Generator{})
	genv1alpha1.RegisterGeneric(genv1alpha1.PostgreSqlKind, &genv1alpha1.PostgreSql{})
}

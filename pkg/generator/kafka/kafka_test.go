package kafka

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/scram"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/testcontainers/testcontainers-go/wait"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

const (
	testNamespace           = "default"
	testSecretName          = "admin-secret"
	testSecretKey           = "password"
	testUser                = "testuser"
	testGeneratedSecretName = "generated-user"
	testPassword            = "adminpass"
	starterScript           = "/usr/sbin/testcontainers_start.sh"
)

type kafkaMockClient struct {
	client.Client
	t            *testing.T
	userPassword []byte
}

func (m kafkaMockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	m.t.Helper()
	if key.Name == testSecretName {
		obj.(*corev1.Secret).Data = map[string][]byte{
			testSecretKey: []byte(testPassword),
		}
	} else if key.Name == testGeneratedSecretName {
		obj.(*corev1.Secret).Data = map[string][]byte{
			testSecretKey: m.userPassword,
		}
	}
	return nil
}

type KafkaTestSuite struct {
	suite.Suite
	ctx      context.Context
	kafka    *tckafka.KafkaContainer
	client   kafkaMockClient
	brokers  []string
	password []byte
}

func TestKafkaTestSuite(t *testing.T) {
	suite.Run(t, new(KafkaTestSuite))
}

func (s *KafkaTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.client = kafkaMockClient{t: s.T()}

	container, err := tckafka.Run(s.ctx,
		"confluentinc/cp-kafka:8.0.0",
		tckafka.WithClusterID("test-cluster"),
		testcontainers.WithEnv(map[string]string{
			// Enable authentication and add SCRAM user
			"KAFKA_LISTENERS":                                              "PLAINTEXT://0.0.0.0:9093,BROKER://0.0.0.0:9092,CONTROLLER://0.0.0.0:9094",
			"KAFKA_REST_BOOTSTRAP_SERVERS":                                 "PLAINTEXT://0.0.0.0:9093,BROKER://0.0.0.0:9092,CONTROLLER://0.0.0.0:9094",
			"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP":                         "PLAINTEXT:SASL_PLAINTEXT,BROKER:PLAINTEXT,CONTROLLER:PLAINTEXT",
			"KAFKA_INTER_BROKER_LISTENER_NAME":                             "BROKER",
			"KAFKA_BROKER_ID":                                              "1",
			"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR":                       "1",
			"KAFKA_OFFSETS_TOPIC_NUM_PARTITIONS":                           "1",
			"KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR":               "1",
			"KAFKA_TRANSACTION_STATE_LOG_MIN_ISR":                          "1",
			"KAFKA_LOG_FLUSH_INTERVAL_MESSAGES":                            strconv.Itoa(math.MaxInt),
			"KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS":                       "0",
			"KAFKA_NODE_ID":                                                "1",
			"KAFKA_PROCESS_ROLES":                                          "broker,controller",
			"KAFKA_CONTROLLER_LISTENER_NAMES":                              "CONTROLLER",
			"KAFKA_ADVERTISED_LISTENERS":                                   "PLAINTEXT",
			"KAFKA_SASL_ENABLED_MECHANISMS":                                "SCRAM-SHA-512",
			"KAFKA_SASL_MECHANISM_INTER_BROKER_PROTOCOL":                   "PLAINTEXT",
			"KAFKA_LISTENER_NAME_PLAINTEXT_SCRAM-SHA-512_SASL_JAAS_CONFIG": `org.apache.kafka.common.security.scram.ScramLoginModule required username="admin" password="password" user_admin="password";`,
			"KAFKA_SASL_JAAS_CONFIG":                                       `org.apache.kafka.common.security.scram.ScramLoginModule required username="admin" password="password";`,
			"KAFKA_SECURITY_PROTOCOL":                                      "SASL_SSL",
			"KAFKA_SASL_MECHANISM":                                         "SCRAM-SHA-512",
			// "KAFKA_SASL_ENABLED_MECHANISMS":                                "PLAIN",
			// "KAFKA_SASL_MECHANISM_INTER_BROKER_PROTOCOL":                   "PLAIN",
			// "KAFKA_LISTENER_NAME_BROKER_PLAIN_SASL_JAAS_CONFIG":            `org.apache.kafka.common.security.plain.PlainLoginModule required username="admin" password="password";`,
			// "KAFKA_LISTENER_NAME_PLAINTEXT_PLAIN_SASL_JAAS_CONFIG":         `org.apache.kafka.common.security.plain.PlainLoginModule required username="admin" password="password";`,
		}),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Kafka Server started").WithStartupTimeout(2*time.Minute),
		),
		// testcontainers.WithCmd(`kafka-storage format --ignore-formatted -t \"\$(kafka-storage random-uuid)\" -c config/server.properties --add-scram 'SCRAM-SHA-256=[name=\"admin\",password=\"password\"]'`),
		testcontainers.WithStartupCommand(testcontainers.NewRawCommand([]string{"echo 'Startup Command'", "ls"})),
		testcontainers.WithAfterReadyCommand(testcontainers.NewRawCommand([]string{"echo 'After Ready Command'", "ls"})),
		testcontainers.WithCmd("-c", "while [ ! -f "+starterScript+" ]; do sleep 0.1; done; echo \"kafka-storage format --ignore-formatted --standalone -t \"$(kafka-storage random-uuid)\" -c /etc/kafka/server.properties --add-scram 'SCRAM-SHA-256=[name=\"admin\",password=\"password\"]'\" >> /etc/confluent/docker/configure; bash "+starterScript),
	)
	container.Exec(s.ctx, []string{"echo 'Exec Command'", "ls"})

	logsReader, logsErr := container.Logs(s.ctx)
	require.NoError(s.T(), logsErr)
	defer logsReader.Close()

	scanner := bufio.NewScanner(logsReader)
	for scanner.Scan() {
		s.T().Logf("Container log: %s", scanner.Text())
	}
	require.NoError(s.T(), scanner.Err())

	require.NoError(s.T(), err)
	s.kafka = container

	isRunning := container.IsRunning()
	require.NoError(s.T(), err)
	s.T().Logf("IsRunning: %v", isRunning)

	state, err := container.State(s.ctx)
	require.NoError(s.T(), err)
	s.T().Logf("State: %+v", state)

	brokers, err := container.Brokers(s.ctx)
	require.NoError(s.T(), err)
	s.T().Logf("Brokers: %v", brokers)
	require.Greater(s.T(), len(brokers), 0)

	inspect, err := container.Inspect(s.ctx)
	require.NoError(s.T(), err)
	ports := inspect.NetworkSettings.Ports
	for port := range ports {
		mapped, err := container.MappedPort(s.ctx, port)
		if err != nil {
			s.T().Logf("Port %v err: %v", port, err)
			continue
		}
		s.T().Logf("Port %v: %v", port, mapped)
	}

	s.brokers = brokers
	s.password = []byte(testPassword)

	s.T().Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	})
}

func (s *KafkaTestSuite) TestGenerateAndCleanupUser() {
	gen := &Generator{}
	username := fmt.Sprintf("%sTest", testUser)

	spec := &genv1alpha1.Kafka{
		Spec: genv1alpha1.KafkaSpec{
			Addresses: s.brokers,
			Auth: genv1alpha1.KafkaAuth{
				Username: "admin",
				Password: genv1alpha1.SecretKeySelector{
					Name: testSecretName,
					Key:  testSecretKey,
				},
			},
			User: &genv1alpha1.KafkaUser{
				Username: username,
				Permissions: []genv1alpha1.KafkaUserPermissions{
					{
						ResourceType:        "topic",
						ResourceName:        "test-topic",
						ResourcePatternType: "literal",
						Operation:           "write",
						PermissionType:      "allow",
					},
				},
			},
		},
	}

	specJSON, err := yaml.Marshal(spec)
	require.NoError(s.T(), err)

	// topic := "my-topic"
	// partition := 0
	// mechanism, err := scram.Mechanism(scram.SHA256, "admin", "password")
	// mechanism := plain.Mechanism{Username: "admin", Password: "password"}

	mechanism, err := scram.Mechanism(scram.SHA512, "admin", "password")
	require.NoError(s.T(), err)
	dialer := &kafka.Dialer{
		Timeout:       10 * time.Second,
		DualStack:     true,
		SASLMechanism: mechanism,
	}

	s.T().Logf("Addr: %s", kafka.TCP(spec.Spec.Addresses...).String())

	addr := kafka.TCP(spec.Spec.Addresses...)
	conn, err := dialer.DialContext(s.ctx, "tcp", addr.String())
	if err != nil {
		s.T().Logf("Dial context err: %v", err)
	}
	if conn != nil {
		require.NoError(s.T(), conn.Close())
	}

	result, statusRaw, err := gen.Generate(s.ctx, &apiextensions.JSON{Raw: specJSON}, s.client, testNamespace)
	require.NoError(s.T(), err)
	require.Contains(s.T(), result, "username")
	require.Contains(s.T(), result, "password")

	generatedUsername := string(result["username"])
	regex := regexp.MustCompile(fmt.Sprintf(`%s_[a-zA-Z0-9]{8}`, username))
	assert.Regexp(s.T(), regex, generatedUsername)

	err = gen.Cleanup(s.ctx, &apiextensions.JSON{Raw: specJSON}, statusRaw, s.client, testNamespace)
	require.NoError(s.T(), err)
}

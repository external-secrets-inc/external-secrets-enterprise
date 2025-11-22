// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Package aws implements AWS authentication.
// Copyright External Secrets Inc. 2025
// All Rights Reserved
package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	modelAWS "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/pkg/models/aws"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/pkg/util"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
)

const (
	// AuthMethodStatic is the static authentication method.
	AuthMethodStatic = "static"
	// AuthMethodIRSA is the IRSA authentication method.
	AuthMethodIRSA = "irsa"
)

// CreateAWSSDKConfig creates an AWS config based on the provided authentication method.
func CreateAWSSDKConfig(ctx context.Context, k8sClient client.Client, config modelAWS.SDKAuth, logger logr.Logger) (aws.Config, error) {
	logger.Info("Creating AWS SDK Config", "AuthMethod", config.AuthMethod)
	switch config.AuthMethod {
	case AuthMethodStatic:
		return loadConfigWithSecret(ctx, k8sClient, config, logger)
	case AuthMethodIRSA:
		return loadConfigWithServiceAccount(ctx, k8sClient, config, logger)
	default:
		err := fmt.Errorf("unsupported authentication method: %s", config.AuthMethod)
		logger.Error(err, "Unsupported authentication method", "AuthMethod", config.AuthMethod)
		return aws.Config{}, err
	}
}

// loadConfigWithSecret loads AWS configuration using static credentials from a Kubernetes Secret.
func loadConfigWithSecret(ctx context.Context, k8sClient client.Client, authConfig modelAWS.SDKAuth, logger logr.Logger) (aws.Config, error) {
	logger.Info("Loading AWS Config with static credentials from secret", "SecretName", authConfig.SecretRef.SecretAccessKey.Name, "Namespace", authConfig.SecretRef.SecretAccessKey.Namespace)
	logger.Info("Loading AWS Config with static credentials from secret", "SecretName", authConfig.SecretRef.AccessKeyID.Name, "Namespace", authConfig.SecretRef.AccessKeyID.Namespace)
	keyIDSecret, err := util.GetSecret(ctx, k8sClient, authConfig.SecretRef.AccessKeyID.Name, authConfig.SecretRef.AccessKeyID.Namespace, logger)
	if err != nil {
		logger.Error(err, "Failed to retrieve secret", "SecretName", authConfig.SecretRef.AccessKeyID.Name, "Namespace", authConfig.SecretRef.AccessKeyID.Namespace)
		return aws.Config{}, err
	}
	accessKeyIDBytes, ok := keyIDSecret.Data[authConfig.SecretRef.AccessKeyID.Key]
	if !ok {
		err := fmt.Errorf("key not found in secret %s", authConfig.SecretRef.AccessKeyID.Name)
		logger.Error(err, "key not found in secret", "SecretName", authConfig.SecretRef.AccessKeyID.Name)
		return aws.Config{}, err
	}
	accessKeySecret, err := util.GetSecret(ctx, k8sClient, authConfig.SecretRef.SecretAccessKey.Name, authConfig.SecretRef.SecretAccessKey.Namespace, logger)
	if err != nil {
		logger.Error(err, "Failed to retrieve secret", "SecretName", authConfig.SecretRef.SecretAccessKey.Name, "Namespace", authConfig.SecretRef.SecretAccessKey.Namespace)
		return aws.Config{}, err
	}

	secretAccessKeyBytes, ok := accessKeySecret.Data[authConfig.SecretRef.SecretAccessKey.Key]
	if !ok {
		err := fmt.Errorf("key not found in secret %s", authConfig.SecretRef.SecretAccessKey.Name)
		logger.Error(err, "key not found in secret", "SecretName", authConfig.SecretRef.SecretAccessKey.Name)
		return aws.Config{}, err
	}
	logger.Info("Successfully retrieved AWS credentials from secret", "SecretName", authConfig.SecretRef.SecretAccessKey.Name)
	return config.LoadDefaultConfig(ctx,
		config.WithRegion(authConfig.Region),
		config.WithCredentialsProvider(awscreds.NewStaticCredentialsProvider(
			string(accessKeyIDBytes),
			string(secretAccessKeyBytes),
			"",
		)),
	)
}

// loadConfigWithServiceAccount loads AWS configuration using IRSA and service account impersonation.
func loadConfigWithServiceAccount(ctx context.Context, k8sClient client.Client, authConfig modelAWS.SDKAuth, logger logr.Logger) (aws.Config, error) {
	logger.Info("Loading AWS Config using IRSA and service account impersonation", "ServiceAccount", authConfig.ServiceAccount.Name, "Namespace", authConfig.ServiceAccount.Namespace)
	// Get service account token
	tokenRetriever := util.NewTokenRetriever(k8sClient, logger, authConfig.ServiceAccount.Name, authConfig.ServiceAccount.Namespace)
	// Get Role ARN from service account annotations
	roleARN, err := getRoleARN(ctx, k8sClient, authConfig.ServiceAccount.Name, authConfig.ServiceAccount.Namespace, logger)
	if err != nil {
		logger.Error(err, "Failed to get Role ARN from service account", "ServiceAccount", authConfig.ServiceAccount.Name)
		return aws.Config{}, err
	}
	logger.Info("Successfully retrieved Role ARN from service account", "RoleARN", roleARN)
	// Create AWS config with the token and role ARN
	return createAWSSessionWithWebIdentity(ctx, tokenRetriever, roleARN, authConfig.Region, logger)
}

// getRoleARN retrieves the IAM Role ARN from the service account annotations.
func getRoleARN(ctx context.Context, k8sClient client.Client, serviceAccountName, namespace string, logger logr.Logger) (string, error) {
	logger.Info("Retrieving Role ARN from service account annotations", "ServiceAccount", serviceAccountName, "Namespace", namespace)
	serviceAccount := &corev1.ServiceAccount{}
	key := types.NamespacedName{
		Name:      serviceAccountName,
		Namespace: namespace,
	}
	if err := k8sClient.Get(ctx, key, serviceAccount); err != nil {
		logger.Error(err, "Failed to get service account", "ServiceAccount", serviceAccountName, "Namespace", namespace)
		return "", fmt.Errorf("failed to get service account: %w", err)
	}
	if serviceAccount.Annotations == nil {
		err := fmt.Errorf("no annotations found on service account %s", serviceAccountName)
		logger.Error(err, "No annotations found on service account", "ServiceAccount", serviceAccountName)
		return "", err
	}
	roleARN, ok := serviceAccount.Annotations["eks.amazonaws.com/role-arn"]
	if !ok {
		err := fmt.Errorf("role ARN annotation not found on service account %s", serviceAccountName)
		logger.Error(err, "Role ARN annotation not found on service account", "ServiceAccount", serviceAccountName)
		return "", err
	}
	logger.Info("Successfully retrieved Role ARN from service account", "RoleARN", roleARN)
	return roleARN, nil
}

// createAWSSessionWithWebIdentity creates an AWS config using Web Identity token and role ARN.
func createAWSSessionWithWebIdentity(ctx context.Context, retriever *util.TokenRetriever, roleARN, region string, logger logr.Logger) (aws.Config, error) {
	logger.Info("Creating AWS session with Web Identity", "RoleARN", roleARN, "Region", region)
	// Create a STS client
	stsClient := sts.New(sts.Options{
		Region: region,
	})
	// Create a WebIdentityRoleProvider with a custom TokenRetriever
	webIdentityProvider := stscreds.NewWebIdentityRoleProvider(stsClient, roleARN, retriever)
	// Create AWS config with the WebIdentityRoleProvider
	awsConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(aws.NewCredentialsCache(webIdentityProvider)),
	)
	if err != nil {
		logger.Error(err, "Failed to load AWS config with Web Identity")
		return aws.Config{}, err
	}
	logger.Info("Successfully created AWS config with Web Identity")
	return awsConfig, nil
}

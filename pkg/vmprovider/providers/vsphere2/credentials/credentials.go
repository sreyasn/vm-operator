// Copyright (c) 2019-2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package credentials

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntime "sigs.k8s.io/controller-runtime/pkg/client"
)

// VSphereVMProviderCredentials wraps the data needed to login to vCenter.
type VSphereVMProviderCredentials struct {
	Username string
	Password string
}

func GetProviderCredentials(client ctrlruntime.Client, namespace, secretName string) (*VSphereVMProviderCredentials, error) {
	secret := &corev1.Secret{}
	secretKey := types.NamespacedName{Namespace: namespace, Name: secretName}
	if err := client.Get(context.Background(), secretKey, secret); err != nil {
		// Log message used by VMC LINT. Refer to before making changes
		return nil, errors.Wrapf(err, "cannot find secret for provider credentials: %s", secretKey)
	}

	var credentials VSphereVMProviderCredentials
	credentials.Username = string(secret.Data["username"])
	credentials.Password = string(secret.Data["password"])

	if credentials.Username == "" || credentials.Password == "" {
		return nil, errors.New("vCenter username and password are missing")
	}

	return &credentials, nil
}

func setSecretData(secret *corev1.Secret, credentials *VSphereVMProviderCredentials) {
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}

	secret.Data["username"] = []byte(credentials.Username)
	secret.Data["password"] = []byte(credentials.Password)
}

// ProviderCredentialsToSecret returns the Secret for the credentials.
// Testing only.
func ProviderCredentialsToSecret(namespace string, credentials *VSphereVMProviderCredentials, vcCredsSecretName string) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vcCredsSecretName,
			Namespace: namespace,
		},
	}
	setSecretData(secret, credentials)

	return secret
}

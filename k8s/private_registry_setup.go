package k8s

import (
	"github.com/cf-platform-eng/kibosh/config"

	"encoding/json"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	api_errors "k8s.io/apimachinery/pkg/api/errors"
)

const registrySecretName = "registry-secret"

type PrivateRegistrySetup interface {
	Setup() error
}

type privateRegistrySetup struct {
	namespace      string
	serviceAccount string
	cluster        Cluster
	registryConfig *config.RegistryConfig
}

func NewPrivateRegistrySetup(namespace string, serviceAccount string, cluster Cluster, registryConfig *config.RegistryConfig) PrivateRegistrySetup {
	return &privateRegistrySetup{
		namespace:      namespace,
		serviceAccount: serviceAccount,
		cluster:        cluster,
		registryConfig: registryConfig,
	}
}

func (p *privateRegistrySetup) Setup() error {
	dockerConfig, _ := p.registryConfig.GetDockerConfigJson()
	secret := &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: registrySecretName,
		},
		Type: api_v1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			api_v1.DockerConfigJsonKey: dockerConfig,
		},
	}
	_, err := p.UpdateOrCreateSecret(p.namespace, secret)

	if err != nil {
		return err
	}

	patch := map[string]interface{}{
		"imagePullSecrets": []map[string]interface{}{
			{"name": registrySecretName},
		},
	}
	patchJson, _ := json.Marshal(patch)
	_, err = p.cluster.Patch(p.namespace, p.serviceAccount, types.MergePatchType, patchJson)
	return err
}

func (p *privateRegistrySetup) UpdateOrCreateSecret(nameSpace string, secret *api_v1.Secret) (*api_v1.Secret, error) {
	_, err := p.cluster.GetSecret(nameSpace, secret.Name, meta_v1.GetOptions{})
	if err != nil {
		statusErr, ok := err.(*api_errors.StatusError)
		if !ok {
			return nil, err
		}
		if statusErr.ErrStatus.Reason == meta_v1.StatusReasonNotFound {
			return p.cluster.CreateSecret(nameSpace, secret)
		} else {
			return nil, err
		}
	}
	return p.cluster.UpdateSecret(nameSpace, secret)
}

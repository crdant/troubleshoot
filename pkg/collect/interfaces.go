package collect

import (
	"github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	corev1 "k8s.io/api/core/v1"
)

type ImageRunner interface {
	GetImage() string
	SetImage(string)

	GetImagePullSecret() *v1beta2.ImagePullSecrets
	SetImagePullSecret(*v1beta2.ImagePullSecrets)

	GetNamespace() string
}

var _ ImageRunner = &v1beta2.Run{}
var _ ImageRunner = &v1beta2.CopyFromHost{}
var _ ImageRunner = &v1beta2.Sysctl{}
var _ ImageRunner = &v1beta2.Collectd{}

type PodSpecRunner interface {
	GetPodSpec() corev1.PodSpec
	SetPodSpec(corev1.PodSpec)

	GetImagePullSecret() *v1beta2.ImagePullSecrets
	SetImagePullSecret(*v1beta2.ImagePullSecrets)

	GetNamespace() string
}

var _ PodSpecRunner = &v1beta2.RunPod{}

// AuthConfigProvider is an interface for collectors that need registry authentication
type AuthConfigProvider interface {
	GetImagePullSecrets() *v1beta2.ImagePullSecrets
	GetNamespace() string
}

// Ensure both registry and image signatures collectors implement this interface
var _ AuthConfigProvider = &v1beta2.RegistryImages{}
var _ AuthConfigProvider = &v1beta2.ImageSignatures{}

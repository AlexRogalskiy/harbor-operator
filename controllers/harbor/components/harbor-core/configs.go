package core

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/markbates/pkger"
	containerregistryv1alpha1 "github.com/ovh/harbor-operator/api/v1alpha1"
	"github.com/ovh/harbor-operator/pkg/factories/application"
	"github.com/pkg/errors"
)

const (
	configName = "app.conf"
)

var (
	once   sync.Once
	config []byte
)

func InitConfigMaps() {
	file, err := pkger.Open("/assets/templates/core/app.conf")
	if err != nil {
		panic(errors.Wrapf(err, "cannot open Core configuration template %s", "/assets/templates/core/app.conf"))
	}
	defer file.Close()

	config, err = ioutil.ReadAll(file)
	if err != nil {
		panic(errors.Wrapf(err, "cannot read Core configuration template %s", "/assets/templates/core/app.conf"))
	}
}

func (c *HarborCore) GetConfigMaps(ctx context.Context) []*corev1.ConfigMap {
	once.Do(InitConfigMaps)

	operatorName := application.GetName(ctx)
	harborName := c.harbor.Name

	return []*corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.harbor.NormalizeComponentName(containerregistryv1alpha1.CoreName),
				Namespace: c.harbor.Namespace,
				Labels: map[string]string{
					"app":      containerregistryv1alpha1.CoreName,
					"harbor":   harborName,
					"operator": operatorName,
				},
			},

			BinaryData: map[string][]byte{
				configName: config,
			},

			// https://github.com/goharbor/harbor/blob/master/make/photon/prepare/templates/core/env.jinja
			Data: map[string]string{
				"CONFIG_PATH": coreConfigPath,

				"AUTH_MODE":                      "db_auth",
				"CFG_EXPIRATION":                 "5",
				"CHART_CACHE_DRIVER":             "memory",
				"EXT_ENDPOINT":                   c.harbor.Spec.PublicURL,
				"LOG_LEVEL":                      "debug",
				"REGISTRY_STORAGE_PROVIDER_NAME": "memory",
				"SYNC_REGISTRY":                  "false",

				"_REDIS_URL":                    "", // For session purpose
				"ADMIRAL_URL":                   "NA",
				"CHART_REPOSITORY_URL":          fmt.Sprintf("http://%s", c.harbor.NormalizeComponentName(containerregistryv1alpha1.ChartMuseumName)),
				"CLAIR_HEALTH_CHECK_SERVER_URL": fmt.Sprintf("http://%s:6061", c.harbor.NormalizeComponentName(containerregistryv1alpha1.ClairName)),
				"CLAIR_URL":                     fmt.Sprintf("http://%s", c.harbor.NormalizeComponentName(containerregistryv1alpha1.ClairName)),
				"CORE_URL":                      fmt.Sprintf("http://%s", c.harbor.NormalizeComponentName(containerregistryv1alpha1.CoreName)),
				"JOBSERVICE_URL":                fmt.Sprintf("http://%s", c.harbor.NormalizeComponentName(containerregistryv1alpha1.JobServiceName)),
				"NOTARY_URL":                    fmt.Sprintf("http://%s", c.harbor.NormalizeComponentName(containerregistryv1alpha1.NotaryServerName)),
				"PORTAL_URL":                    fmt.Sprintf("http://%s", c.harbor.NormalizeComponentName(containerregistryv1alpha1.PortalName)),
				"REGISTRY_URL":                  fmt.Sprintf("http://%s", c.harbor.NormalizeComponentName(containerregistryv1alpha1.RegistryName)),
				"REGISTRYCTL_URL":               fmt.Sprintf("http://%s:8080", c.harbor.NormalizeComponentName(containerregistryv1alpha1.RegistryName)),
				"TOKEN_SERVICE_URL":             fmt.Sprintf("http://%s/service/token", c.harbor.NormalizeComponentName(containerregistryv1alpha1.CoreName)),

				"DATABASE_TYPE":             "postgresql",
				"POSTGRESQL_MAX_IDLE_CONNS": fmt.Sprintf("%d", maxIdleConns),
				"POSTGRESQL_MAX_OPEN_CONNS": fmt.Sprintf("%d", maxOpenConns),

				"WITH_CHARTMUSEUM": strconv.FormatBool(c.harbor.Spec.Components.ChartMuseum != nil),
				"WITH_CLAIR":       strconv.FormatBool(c.harbor.Spec.Components.Clair != nil),
				"WITH_NOTARY":      strconv.FormatBool(c.harbor.Spec.Components.Notary != nil),
			},
		},
	}
}

func (c *HarborCore) GetConfigCheckSum() string {
	checksum := strings.Join([]string{
		c.harbor.Spec.PublicURL,
		strconv.FormatBool(c.harbor.Spec.Components.Clair != nil),
	}, "\n")

	h := sha256.New()

	return fmt.Sprintf("%x", h.Sum([]byte(checksum)))
}

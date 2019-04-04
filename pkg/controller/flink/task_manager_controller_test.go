package flink

import (
	k8mock "github.com/lyft/flinkk8soperator/pkg/controller/k8/mock"
	mockScope "github.com/lyft/flytestdlib/promutils"
	"testing"

	"context"
	"github.com/lyft/flinkk8soperator/pkg/apis/app/v1alpha1"
	"github.com/lyft/flinkk8soperator/pkg/controller/common"
	"github.com/lyft/flytestdlib/promutils/labeled"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func getTMControllerForTest() FlinkTaskManagerController {
	testScope := mockScope.NewTestScope()
	labeled.SetMetricKeys(common.GetValidLabelNames()...)

	return FlinkTaskManagerController{
		metrics:   newFlinkTaskManagerMetrics(testScope),
		k8Cluster: &k8mock.MockK8Cluster{},
	}
}

func TestComputeTaskManagerReplicas(t *testing.T) {
	app := v1alpha1.FlinkApplication{}
	taskSlots := int32(4)
	app.Spec.TaskManagerConfig.TaskSlots = &taskSlots
	app.Spec.FlinkJob.Parallelism = 9

	assert.Equal(t, int32(3), computeTaskManagerReplicas(&app))
}

func TestGetTaskManagerName(t *testing.T) {
	app := getFlinkTestApp()
	assert.Equal(t, "app-name-11ae1-tm", getTaskManagerName(app))
}

func TestGetTaskManagerPodName(t *testing.T) {
	app := getFlinkTestApp()
	assert.Equal(t, "app-name-11ae1-tm-pod", getTaskManagerPodName(app))
}

func TestGetTaskManagerDeployment(t *testing.T) {
	app := getFlinkTestApp()
	deployment := v1.Deployment{}
	deployment.Name = getTaskManagerName(app)
	deployments := []v1.Deployment{
		deployment,
	}
	assert.Equal(t, deployment, *getTaskManagerDeployment(deployments, &app))
}

func TestGetTaskManagerReplicaCount(t *testing.T) {
	app := getFlinkTestApp()
	deployment := v1.Deployment{}
	deployment.Name = getTaskManagerName(app)
	replicaCount := int32(2)
	deployment.Spec.Replicas = &replicaCount
	deployments := []v1.Deployment{
		deployment,
	}
	assert.Equal(t, int32(2), getTaskManagerCount(deployments, &app))
}

func TestTaskManagerCreateSuccess(t *testing.T) {
	testController := getTMControllerForTest()
	app := getFlinkTestApp()
	annotations := map[string]string{
		"key": "annotation",
	}
	app.Annotations = annotations
	expectedLabels := map[string]string{
		"app":      "app-name",
		"imageKey": "11ae1",
	}
	mockK8Cluster := testController.k8Cluster.(*k8mock.MockK8Cluster)
	mockK8Cluster.CreateK8ObjectFunc = func(ctx context.Context, object sdk.Object) error {
		deployment := object.(*v1.Deployment)
		assert.Equal(t, getTaskManagerName(app), deployment.Name)
		assert.Equal(t, app.Namespace, deployment.Namespace)
		assert.Equal(t, getTaskManagerPodName(app), deployment.Spec.Template.Name)
		assert.Equal(t, annotations, deployment.Annotations)
		assert.Equal(t, annotations, deployment.Spec.Template.Annotations)
		assert.Equal(t, app.Namespace, deployment.Spec.Template.Namespace)
		assert.Equal(t, expectedLabels, deployment.Labels)

		return nil
	}
	err := testController.CreateIfNotExist(context.Background(), &app)
	assert.Nil(t, err)
}

func TestTaskManagerCreateErr(t *testing.T) {
	testController := getTMControllerForTest()
	app := getFlinkTestApp()
	mockK8Cluster := testController.k8Cluster.(*k8mock.MockK8Cluster)
	mockK8Cluster.CreateK8ObjectFunc = func(ctx context.Context, object sdk.Object) error {
		return errors.New("create error")
	}
	err := testController.CreateIfNotExist(context.Background(), &app)
	assert.EqualError(t, err, "create error")
}

func TestTaskManagerCreateAlreadyExists(t *testing.T) {
	testController := getTMControllerForTest()
	app := getFlinkTestApp()
	mockK8Cluster := testController.k8Cluster.(*k8mock.MockK8Cluster)
	mockK8Cluster.CreateK8ObjectFunc = func(ctx context.Context, object sdk.Object) error {
		return k8sErrors.NewAlreadyExists(schema.GroupResource{}, "")
	}
	err := testController.CreateIfNotExist(context.Background(), &app)
	assert.Nil(t, err)
}
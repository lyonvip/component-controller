/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"component-controller/controllers/utils"
	"context"
	"fmt"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	compv1 "component-controller/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// ComponentReconciler reconciles a Component object
type ComponentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=comp.base.io,resources=components,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=comp.base.io,resources=components/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=comp.base.io,resources=components/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Component object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ComponentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	/* 获取缓存中component对象 */
	cacheComponent := &compv1.Component{}
	if err := r.Get(ctx, req.NamespacedName, cacheComponent); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cacheComponentName := cacheComponent.Name + "." + cacheComponent.Namespace

	/* 处理configmap */
	nonConfigmapList := []string{"kafka", "mongodb"}
	if !slice.Contain(nonConfigmapList, cacheComponent.Spec.Type) {
		// 创建configmap并与component资源关联
		genConfigmap := utils.NewConfigmap(cacheComponent)
		if err := controllerutil.SetControllerReference(cacheComponent, genConfigmap, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		// 获取缓存中的configmap
		cacheConfigmap := &corev1.ConfigMap{}
		cmName := fmt.Sprintf("%s-config", cacheComponent.Spec.Type)
		if err := r.Get(ctx, types.NamespacedName{Name: cmName, Namespace: cacheComponent.Namespace}, cacheConfigmap); err != nil {
			if errors.IsNotFound(err) {
				// 创建configmap
				if err = r.Create(ctx, genConfigmap); err != nil {
					logger.Error(err, fmt.Sprintf("[%s] create new configmap failed", cacheComponentName))
					return ctrl.Result{}, err
				}
			} else {
				return ctrl.Result{}, err
			}
		}
	}

	/* 处理secret */
	// 创建secret并于component资源相关联
	genSecret := utils.NewSecret(cacheComponent)
	if err := controllerutil.SetControllerReference(cacheComponent, genSecret, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	// 获取缓存中的secret
	cacheSecret := &corev1.Secret{}
	authName := fmt.Sprintf("%s-auth", cacheComponent.Spec.Type)
	if err := r.Get(ctx, types.NamespacedName{Name: authName, Namespace: cacheComponent.Namespace}, cacheSecret); err != nil {
		if errors.IsNotFound(err) {
			// 创建secret
			if err = r.Create(ctx, genSecret); err != nil {
				logger.Error(err, fmt.Sprintf("[%s] create new secret failed", cacheComponentName))
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	}

	/* 处理deployment */
	// 创建deployment并与component资源相关联
	genDeployment := utils.NewDeployment(cacheComponent)
	if err := controllerutil.SetControllerReference(cacheComponent, genDeployment, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	// 获取缓存中的deployment
	cacheDeployment := &appsv1.Deployment{}
	deployName := fmt.Sprintf("%s-server", cacheComponent.Spec.Type)
	if err := r.Get(ctx, types.NamespacedName{Name: deployName, Namespace: cacheComponent.Namespace}, cacheDeployment); err != nil {
		if errors.IsNotFound(err) {
			// 创建deployment
			if err = r.Create(ctx, genDeployment); err != nil {
				logger.Error(err, fmt.Sprintf("[%s] create new deployment failed", cacheComponentName))
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	} else {
		// 更新deployment
		if updateDeploymentWhenChange(cacheComponent.Spec.Type, cacheDeployment, genDeployment) {
			*cacheDeployment.Spec.Replicas = 0
			if err = r.Update(ctx, cacheDeployment); err != nil {
				logger.Error(err, fmt.Sprintf("[%s] scale deployment %s to zero failed", cacheComponentName, cacheComponent.Spec.Type+"-server"))
				return ctrl.Result{}, err
			}
			if err = r.Update(ctx, genDeployment); err != nil {
				logger.Error(err, fmt.Sprintf("[%s] update deployment %s failed", cacheComponentName, cacheComponent.Spec.Type+"-server"))
				return ctrl.Result{}, err
			}
		}
	}

	/* 处理service */
	// 创建service并与component资源相关联
	genService := utils.NewService(cacheComponent)
	if err := controllerutil.SetControllerReference(cacheComponent, genService, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	// 获取缓存中的service
	cacheService := &corev1.Service{}
	svcName := fmt.Sprintf("%s-server-svc", cacheComponent.Spec.Type)
	if err := r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: cacheComponent.Namespace}, cacheService); err != nil {
		if errors.IsNotFound(err) {
			// 创建service
			if err = r.Create(ctx, genService); err != nil {
				logger.Error(err, fmt.Sprintf("[%s] create new service failed", cacheComponentName))
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	} else {
		// 更新service
		if updateServiceWhenChange(cacheService, genService) {
			if err = r.Update(ctx, genService); err != nil {
				logger.Error(err, fmt.Sprintf("[%s] update service %s failed", cacheComponentName, svcName))
				return ctrl.Result{}, err
			}
		}
	}

	/* 更新component资源的status状态 */
	// 获取IsValidate状态
	if err := updateComponentStatus(ctx, r, cacheComponent, genService, genSecret, logger); err != nil {
		logger.Error(err, fmt.Sprintf("[%s] update status failed", cacheComponentName))
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ComponentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&compv1.Component{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func isEqualMap(m1, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k, v := range m1 {
		if v != m2[k] {
			return false
		}
	}
	return true
}

func updateDeploymentWhenChange(compType string, cacheDeployment, genDeployment *appsv1.Deployment) bool {
	/* 获取缓存deployment的server容器、sidecar容器 */
	/* 获取期望deployment的server容器、sidecar容器 */
	var cacheServer, genServer, cacheSidecar, genSidecar corev1.Container
	nonSidecarCompList := []string{"rabbitmq"}
	cacheContainerList := cacheDeployment.Spec.Template.Spec.Containers
	genContainerList := genDeployment.Spec.Template.Spec.Containers
	for _, value := range cacheContainerList {
		if value.Name == compType+"-server" {
			cacheServer = value
		}
		if !slice.Contain(nonSidecarCompList, compType) {
			if value.Name == compType+"-sidecar" {
				cacheSidecar = value
			}
		}
	}
	for _, value := range genContainerList {
		if value.Name == compType+"-server" {
			genServer = value
		}
		if !slice.Contain(nonSidecarCompList, compType) {
			if value.Name == compType+"-sidecar" {
				genSidecar = value
			}
		}
	}
	/* 判断nodeSelector是否变化 */
	if !isEqualMap(genDeployment.Spec.Template.Spec.NodeSelector, cacheDeployment.Spec.Template.Spec.NodeSelector) {
		return true
	}
	/* 判断server容器是否变化 */
	if !(cacheServer.Image == genServer.Image && reflect.DeepEqual(cacheServer.Resources, genServer.Resources)) {
		return true
	}
	/* 判断sidecar容器是否变化 */
	if !slice.Contain(nonSidecarCompList, compType) {
		if cacheSidecar.Image != genSidecar.Image {
			return true
		}
	}
	return false
}

func updateServiceWhenChange(cacheService *corev1.Service, genService *corev1.Service) bool {
	if cacheService.Spec.Type != genService.Spec.Type {
		return true
	}
	return false
}

func updateComponentStatus(ctx context.Context, r *ComponentReconciler, cacheComponent *compv1.Component, genService *corev1.Service, genSecret *corev1.Secret, logger logr.Logger) error {
	if cacheComponent.Status.IsValidate == nil {
		cacheComponent.Status.IsValidate = new(bool)
		*cacheComponent.Status.IsValidate = false
	} else {
		status, err := utils.GetCompStatus(cacheComponent.Spec.Type, genService, genSecret)
		if err != nil {
			logger.Error(err, "get component status failed")
			return err
		}
		*cacheComponent.Status.IsValidate = status
	}

	if err := r.Status().Update(ctx, cacheComponent); err != nil {
		return err
	}
	return nil
}

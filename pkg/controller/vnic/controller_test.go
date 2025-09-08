package vnic

import (
	"context"
	"fmt"
	"reflect"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/everoute/trafficredirect/api/trafficredirect/v1alpha1"
	"github.com/everoute/trafficredirect/pkg/constants"
	"github.com/everoute/trafficredirect/pkg/tower/datamodel"
	"github.com/everoute/trafficredirect/pkg/tower/client"
)

// mockStatusWriter 实现 client.StatusWriter 接口
type mockStatusWriter struct {
	client *mockK8sClient
}

func (m *mockStatusWriter) Update(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.SubResourceUpdateOption) error {
	return m.client.Update(ctx, obj)
}

func (m *mockStatusWriter) Patch(ctx context.Context, obj k8sclient.Object, patch k8sclient.Patch, opts ...k8sclient.SubResourcePatchOption) error {
	// 简化实现
	return nil
}

func (m *mockStatusWriter) Create(ctx context.Context, obj k8sclient.Object, subResource k8sclient.Object, opts ...k8sclient.SubResourceCreateOption) error {
	return fmt.Errorf("not implemented")
}

// mock Kubernetes client
type mockK8sClient struct {
	k8sclient.Client
	getError    error
	createError error
	updateError error
	deleteError error
	rules       map[types.NamespacedName]*v1alpha1.Rule
}

func newMockK8sClient() *mockK8sClient {
	return &mockK8sClient{
		rules: make(map[types.NamespacedName]*v1alpha1.Rule),
	}
}

func (m *mockK8sClient) Get(ctx context.Context, key k8sclient.ObjectKey, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
	if m.getError != nil {
		return m.getError
	}

	rule, ok := obj.(*v1alpha1.Rule)
	if !ok {
		return fmt.Errorf("unexpected object type")
	}

	if existingRule, exists := m.rules[key]; exists {
		*rule = *existingRule
		return nil
	}

	return apierrors.NewNotFound(schema.GroupResource{}, key.Name)
}

func (m *mockK8sClient) Create(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.CreateOption) error {
	if m.createError != nil {
		return m.createError
	}

	rule, ok := obj.(*v1alpha1.Rule)
	if !ok {
		return fmt.Errorf("unexpected object type")
	}

	key := types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}
	if _, exists := m.rules[key]; exists {
		return apierrors.NewAlreadyExists(schema.GroupResource{}, rule.Name)
	}

	m.rules[key] = rule.DeepCopy()
	return nil
}

func (m *mockK8sClient) Update(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.UpdateOption) error {
	if m.updateError != nil {
		return m.updateError
	}

	rule, ok := obj.(*v1alpha1.Rule)
	if !ok {
		return fmt.Errorf("unexpected object type")
	}

	key := types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}
	if _, exists := m.rules[key]; !exists {
		return apierrors.NewNotFound(schema.GroupResource{}, rule.Name)
	}

	m.rules[key] = rule.DeepCopy()
	return nil
}

func (m *mockK8sClient) Delete(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.DeleteOption) error {
	if m.deleteError != nil {
		return m.deleteError
	}

	rule, ok := obj.(*v1alpha1.Rule)
	if !ok {
		return fmt.Errorf("unexpected object type")
	}

	key := types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}
	if _, exists := m.rules[key]; !exists {
		return apierrors.NewNotFound(schema.GroupResource{}, rule.Name)
	}

	delete(m.rules, key)
	return nil
}

func (m *mockK8sClient) Scheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	return scheme
}

func (m *mockK8sClient) Status() k8sclient.StatusWriter {
	return &mockStatusWriter{client: m}
}

func (m *mockK8sClient) Patch(ctx context.Context, obj k8sclient.Object, patch k8sclient.Patch, opts ...k8sclient.PatchOption) error {
	return nil
}

func (m *mockK8sClient) DeleteAllOf(ctx context.Context, obj k8sclient.Object, opts ...k8sclient.DeleteAllOfOption) error {
	return nil
}

// 辅助函数：创建测试用的 Rule 对象
func createTestRule(name, direct, srcMac, dstMac, vmID, nic string) *v1alpha1.Rule {
	return &v1alpha1.Rule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: constants.VnicRuleNamespace,
		},
		Spec: v1alpha1.RuleSpec{
			Direct: v1alpha1.RuleDirect(direct),
			Match: v1alpha1.RuleMatch{
				SrcMac: srcMac,
				DstMac: dstMac,
			},
			Option: &v1alpha1.Option{
				TowerVM: vmID,
			},
		},
	}
}

var _ = Describe("Vnic Controller", func() {
	var (
		c          *Controller
		ctx        context.Context
		mockClient *mockK8sClient
		patches    *gomonkey.Patches
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockClient = newMockK8sClient()
	})

	AfterEach(func() {
		if c != nil && c.queue != nil {
			c.queue.ShutDown()
		}
		if patches != nil {
			patches.Reset()
		}
	})

	Context("handle function with Rule CRD (gomonkey towerCli.Get)", func() {
		var towerCli *client.Client

		BeforeEach(func() {
			mockClient = newMockK8sClient()
			mockClient.rules = make(map[types.NamespacedName]*v1alpha1.Rule)
			towerCli = &client.Client{}
			c = &Controller{
				k8scli:   mockClient,
				towerCli: towerCli,
				queue:    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
			}
		})

		It("should handle vnic not found in tower", func() {
			mockClient.rules[types.NamespacedName{Namespace: constants.VnicRuleNamespace, Name: vnicIDToRuleName("vnic1", v1alpha1.Ingress)}] = createTestRule(vnicIDToRuleName("vnic1", v1alpha1.Ingress), string(v1alpha1.Ingress), "", "aa:bb:cc:dd:ee:ff", "vm1", "vnic1")
			mockClient.rules[types.NamespacedName{Namespace: constants.VnicRuleNamespace, Name: vnicIDToRuleName("vnic1", v1alpha1.Egress)}] = createTestRule(vnicIDToRuleName("vnic1", v1alpha1.Egress), string(v1alpha1.Egress), "aa:bb:cc:dd:ee:ff", "", "vm1", "vnic1")

			patches = gomonkey.ApplyMethod(reflect.TypeOf(towerCli), "Get",
				func(_ *client.Client, _ context.Context, id string, vnic datamodel.GqlType) (bool, error) {
					return false, nil
				},
			)
			err := c.handle(ctx, "vnic1")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(mockClient.rules)).To(Equal(0))
		})

		It("should handle towerCli.Get error", func() {
			patches = gomonkey.ApplyMethod(reflect.TypeOf(towerCli), "Get",
				func(_ *client.Client, _ context.Context, id string, vnic datamodel.GqlType) (bool, error) {
					return false, fmt.Errorf("get error")
				},
			)
			err := c.handle(ctx, "vnic1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("get error"))
		})

		It("should create rules with correct CRD structure when DPI is enabled", func() {
			patches = gomonkey.ApplyMethod(reflect.TypeOf(towerCli), "Get",
				func(_ *client.Client, _ context.Context, id string, v datamodel.GqlType) (bool, error) {
					vnic := v.(*datamodel.VMNic)
					vnic.ID = "vnic1"
					vnic.DPIEnabled = true
					vnic.VM.ID = "vm1"
					vnic.MacAddress = "aa:bb:cc:dd:ee:ff"
					return true, nil
				},
			)
			err := c.handle(ctx, "vnic1")
			Expect(err).NotTo(HaveOccurred())

			// 验证 ingress rule
			ingressKey := types.NamespacedName{
				Namespace: constants.VnicRuleNamespace,
				Name:      vnicIDToRuleName("vnic1", v1alpha1.Ingress),
			}
			Expect(mockClient.rules).To(HaveKey(ingressKey))
			ingressRule := mockClient.rules[ingressKey]
			Expect(ingressRule.Spec.Direct).To(Equal(v1alpha1.Ingress))
			Expect(ingressRule.Spec.Match.SrcMac).To(Equal(""))
			Expect(ingressRule.Spec.Match.DstMac).To(Equal("aa:bb:cc:dd:ee:ff"))
			Expect(ingressRule.Spec.Option.TowerVM).To(Equal("vm1"))

			// 验证 egress rule
			egressKey := types.NamespacedName{
				Namespace: constants.VnicRuleNamespace,
				Name:      vnicIDToRuleName("vnic1", v1alpha1.Egress),
			}
			Expect(mockClient.rules).To(HaveKey(egressKey))
			egressRule := mockClient.rules[egressKey]
			Expect(egressRule.Spec.Direct).To(Equal(v1alpha1.Egress))
			Expect(egressRule.Spec.Match.SrcMac).To(Equal("aa:bb:cc:dd:ee:ff"))
			Expect(egressRule.Spec.Match.DstMac).To(Equal(""))
			Expect(egressRule.Spec.Option.TowerVM).To(Equal("vm1"))
		})

		It("should update existing rules with new spec when DPI settings change", func() {
			// 第一次创建
			patches = gomonkey.ApplyMethod(reflect.TypeOf(towerCli), "Get",
				func(_ *client.Client, _ context.Context, id string, vnic datamodel.GqlType) (bool, error) {
					v := vnic.(*datamodel.VMNic)
					v.ID = "vnic1"
					v.DPIEnabled = true
					v.VM.ID = "vm1"
					v.MacAddress = "aa:bb:cc:dd:ee:ff"
					return true, nil
				},
			)
			err := c.handle(ctx, "vnic1")
			Expect(err).NotTo(HaveOccurred())

			patches.Reset()
			patches = gomonkey.ApplyMethod(reflect.TypeOf(towerCli), "Get",
				func(_ *client.Client, _ context.Context, id string, v datamodel.GqlType) (bool, error) {
					vnic := v.(*datamodel.VMNic)
					vnic.ID = "vnic1"
					vnic.DPIEnabled = true
					vnic.VM.ID = "vm2"
					vnic.MacAddress = "ff:ee:dd:cc:bb:aa"
					return true, nil
				},
			)
			err = c.handle(ctx, "vnic1")
			Expect(err).NotTo(HaveOccurred())

			// 验证规则已更新
			ingressKey := types.NamespacedName{
				Namespace: constants.VnicRuleNamespace,
				Name:      vnicIDToRuleName("vnic1", v1alpha1.Ingress),
			}
			ingressRule := mockClient.rules[ingressKey]
			Expect(ingressRule.Spec.Match.DstMac).To(Equal("ff:ee:dd:cc:bb:aa"))
			Expect(ingressRule.Spec.Option.TowerVM).To(Equal("vm2"))

			egressKey := types.NamespacedName{
				Namespace: constants.VnicRuleNamespace,
				Name:      vnicIDToRuleName("vnic1", v1alpha1.Egress),
			}
			egressRule := mockClient.rules[egressKey]
			Expect(egressRule.Spec.Match.SrcMac).To(Equal("ff:ee:dd:cc:bb:aa"))
			Expect(egressRule.Spec.Option.TowerVM).To(Equal("vm2"))
		})

		It("should not update rules when spec is identical", func() {
			patches = gomonkey.ApplyMethod(reflect.TypeOf(towerCli), "Get",
				func(_ *client.Client, _ context.Context, id string, v datamodel.GqlType) (bool, error) {
					vnic := v.(*datamodel.VMNic)
					vnic.ID = "vnic1"
					vnic.DPIEnabled = true
					vnic.VM.ID = "vm1"
					vnic.MacAddress = "aa:bb:cc:dd:ee:ff"
					return true, nil
				},
			)
			// 第一次处理
			err := c.handle(ctx, "vnic1")
			Expect(err).NotTo(HaveOccurred())

			// 记录原始规则
			ingressKey := types.NamespacedName{
				Namespace: constants.VnicRuleNamespace,
				Name:      vnicIDToRuleName("vnic1", v1alpha1.Ingress),
			}
			originalIngressRule := mockClient.rules[ingressKey].DeepCopy()

			// 第二次处理（相同配置）
			err = c.handle(ctx, "vnic1")
			Expect(err).NotTo(HaveOccurred())

			// 验证规则未改变
			currentIngressRule := mockClient.rules[ingressKey]
			Expect(currentIngressRule.Spec).To(Equal(originalIngressRule.Spec))
		})

		It("should delete rules when DPI is disabled", func() {
			// 先创建规则
			patches = gomonkey.ApplyMethod(reflect.TypeOf(towerCli), "Get",
				func(_ *client.Client, _ context.Context, id string, v datamodel.GqlType) (bool, error) {
					vnic := v.(*datamodel.VMNic)
					vnic.ID = "vnic1"
					vnic.DPIEnabled = true
					vnic.VM.ID = "vm1"
					vnic.MacAddress = "aa:bb:cc:dd:ee:ff"
					return true, nil
				},
			)
			err := c.handle(ctx, "vnic1")
			Expect(err).NotTo(HaveOccurred())

			patches.Reset()
			patches = gomonkey.ApplyMethod(reflect.TypeOf(towerCli), "Get",
				func(_ *client.Client, _ context.Context, id string, v datamodel.GqlType) (bool, error) {
					vnic := v.(*datamodel.VMNic)
					vnic.ID = "vnic1"
					vnic.DPIEnabled = false
					vnic.VM.ID = "vm1"
					vnic.MacAddress = "aa:bb:cc:dd:ee:ff"
					return true, nil
				},
			)
			err = c.handle(ctx, "vnic1")
			Expect(err).NotTo(HaveOccurred())

			// 验证规则已删除
			ingressKey := types.NamespacedName{
				Namespace: constants.VnicRuleNamespace,
				Name:      vnicIDToRuleName("vnic1", v1alpha1.Ingress),
			}
			egressKey := types.NamespacedName{
				Namespace: constants.VnicRuleNamespace,
				Name:      vnicIDToRuleName("vnic1", v1alpha1.Egress),
			}
			_, ingressExists := mockClient.rules[ingressKey]
			_, egressExists := mockClient.rules[egressKey]
			Expect(ingressExists).To(BeFalse())
			Expect(egressExists).To(BeFalse())
		})
	})
	Context("ruleHandle function", func() {
		BeforeEach(func() {
			c = &Controller{
				queue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
			}
		})

		It("should skip non-vnic rules", func() {
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "other-namespace",
					Name:      "some-rule",
				},
			}
			result, err := c.ruleHandle(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(c.queue.Len()).To(Equal(0))
		})

		It("should skip invalid rule names", func() {
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: constants.VnicRuleNamespace,
					Name:      "invalid-rule-name",
				},
			}
			result, err := c.ruleHandle(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(c.queue.Len()).To(Equal(0))
		})

		It("should add vnic to queue for valid vnic rule", func() {
			vnicID := "vnic1"
			ruleName := vnicIDToRuleName(vnicID, v1alpha1.Ingress)
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: constants.VnicRuleNamespace,
					Name:      ruleName,
				},
			}
			result, err := c.ruleHandle(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			item, shutdown := c.queue.Get()
			Expect(shutdown).To(BeFalse())
			Expect(item).To(Equal(vnicID))
			c.queue.Done(item)
		})
	})

	Context("deleteRule function", func() {
		BeforeEach(func() {
			c = &Controller{
				k8scli: mockClient,
			}
		})

		It("should handle not found error gracefully", func() {
			err := c.deleteRule(ctx, "non-existent-rule")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error on other get errors", func() {
			mockClient.getError = fmt.Errorf("get error")
			err := c.deleteRule(ctx, "some-rule")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("get error"))
		})

		It("should return error on delete errors", func() {
			// 先创建一条规则
			rule := createTestRule("test-rule", string(v1alpha1.Ingress), "", "aa:bb:cc:dd:ee:ff", "vm1", "vnic1")
			mockClient.rules[types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}] = rule

			mockClient.deleteError = fmt.Errorf("delete error")
			err := c.deleteRule(ctx, "test-rule")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("delete error"))
		})
	})

	Context("addOrUpdateRule function", func() {
		BeforeEach(func() {
			c = &Controller{
				k8scli: mockClient,
			}
		})

		It("should create new rule when not exists", func() {
			rule := createTestRule("new-rule", string(v1alpha1.Ingress), "", "aa:bb:cc:dd:ee:ff", "vm1", "vnic1")
			err := c.addOrUpdateRule(ctx, rule)
			Expect(err).NotTo(HaveOccurred())
			Expect(mockClient.rules).To(HaveKey(types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}))
		})

		It("should return error on create failure", func() {
			rule := createTestRule("new-rule", string(v1alpha1.Ingress), "", "aa:bb:cc:dd:ee:ff", "vm1", "vnic1")
			mockClient.createError = fmt.Errorf("create error")
			err := c.addOrUpdateRule(ctx, rule)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("create error"))
		})

		It("should update existing rule when spec changes", func() {
			// 先创建规则
			rule := createTestRule("existing-rule", string(v1alpha1.Ingress), "", "aa:bb:cc:dd:ee:ff", "vm1", "vnic1")
			mockClient.rules[types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}] = rule.DeepCopy()

			// 修改规则
			newRule := rule.DeepCopy()
			newRule.Spec.Match.DstMac = "ff:ee:dd:cc:bb:aa"

			err := c.addOrUpdateRule(ctx, newRule)
			Expect(err).NotTo(HaveOccurred())
			Expect(mockClient.rules[types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}].Spec.Match.DstMac).To(Equal("ff:ee:dd:cc:bb:aa"))
		})

		It("should not update rule when spec is identical", func() {
			// 先创建规则
			rule := createTestRule("existing-rule", string(v1alpha1.Ingress), "", "aa:bb:cc:dd:ee:ff", "vm1", "vnic1")
			mockClient.rules[types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}] = rule.DeepCopy()

			err := c.addOrUpdateRule(ctx, rule.DeepCopy())
			Expect(err).NotTo(HaveOccurred())
			// 规则应该保持不变
		})

		It("should return error on get failure", func() {
			rule := createTestRule("some-rule", string(v1alpha1.Ingress), "", "aa:bb:cc:dd:ee:ff", "vm1", "vnic1")
			mockClient.getError = fmt.Errorf("get error")
			err := c.addOrUpdateRule(ctx, rule)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("get error"))
		})

		It("should return error on update failure", func() {
			// 先创建规则
			rule := createTestRule("existing-rule", string(v1alpha1.Ingress), "", "aa:bb:cc:dd:ee:ff", "vm1", "vnic1")
			mockClient.rules[types.NamespacedName{Namespace: rule.Namespace, Name: rule.Name}] = rule.DeepCopy()

			// 修改规则并设置更新错误
			newRule := rule.DeepCopy()
			newRule.Spec.Match.DstMac = "ff:ee:dd:cc:bb:aa"
			mockClient.updateError = fmt.Errorf("update error")

			err := c.addOrUpdateRule(ctx, newRule)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("update error"))
		})
	})
})

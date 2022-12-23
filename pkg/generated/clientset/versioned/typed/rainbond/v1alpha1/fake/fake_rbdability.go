// RAINBOND, Application Management Platform
// Copyright (C) 2014-2021 Goodrain Co., Ltd.

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Rainbond,
// one or multiple Commercial Licenses authorized by Goodrain Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/goodrain/rainbond/pkg/apis/rainbond/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeRBDAbilities implements RBDAbilityInterface
type FakeRBDAbilities struct {
	Fake *FakeRainbondV1alpha1
	ns   string
}

var rbdabilitiesResource = schema.GroupVersionResource{Group: "rainbond.io", Version: "v1alpha1", Resource: "rbdabilities"}

var rbdabilitiesKind = schema.GroupVersionKind{Group: "rainbond.io", Version: "v1alpha1", Kind: "RBDAbility"}

// Get takes name of the rBDAbility, and returns the corresponding rBDAbility object, and an error if there is any.
func (c *FakeRBDAbilities) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.RBDAbility, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(rbdabilitiesResource, c.ns, name), &v1alpha1.RBDAbility{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.RBDAbility), err
}

// List takes label and field selectors, and returns the list of RBDAbilities that match those selectors.
func (c *FakeRBDAbilities) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.RBDAbilityList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(rbdabilitiesResource, rbdabilitiesKind, c.ns, opts), &v1alpha1.RBDAbilityList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.RBDAbilityList{ListMeta: obj.(*v1alpha1.RBDAbilityList).ListMeta}
	for _, item := range obj.(*v1alpha1.RBDAbilityList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested rBDAbilities.
func (c *FakeRBDAbilities) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(rbdabilitiesResource, c.ns, opts))

}

// Create takes the representation of a rBDAbility and creates it.  Returns the server's representation of the rBDAbility, and an error, if there is any.
func (c *FakeRBDAbilities) Create(ctx context.Context, rBDAbility *v1alpha1.RBDAbility, opts v1.CreateOptions) (result *v1alpha1.RBDAbility, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(rbdabilitiesResource, c.ns, rBDAbility), &v1alpha1.RBDAbility{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.RBDAbility), err
}

// Update takes the representation of a rBDAbility and updates it. Returns the server's representation of the rBDAbility, and an error, if there is any.
func (c *FakeRBDAbilities) Update(ctx context.Context, rBDAbility *v1alpha1.RBDAbility, opts v1.UpdateOptions) (result *v1alpha1.RBDAbility, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(rbdabilitiesResource, c.ns, rBDAbility), &v1alpha1.RBDAbility{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.RBDAbility), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeRBDAbilities) UpdateStatus(ctx context.Context, rBDAbility *v1alpha1.RBDAbility, opts v1.UpdateOptions) (*v1alpha1.RBDAbility, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(rbdabilitiesResource, "status", c.ns, rBDAbility), &v1alpha1.RBDAbility{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.RBDAbility), err
}

// Delete takes name of the rBDAbility and deletes it. Returns an error if one occurs.
func (c *FakeRBDAbilities) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(rbdabilitiesResource, c.ns, name, opts), &v1alpha1.RBDAbility{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeRBDAbilities) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(rbdabilitiesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.RBDAbilityList{})
	return err
}

// Patch applies the patch and returns the patched rBDAbility.
func (c *FakeRBDAbilities) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.RBDAbility, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(rbdabilitiesResource, c.ns, name, pt, data, subresources...), &v1alpha1.RBDAbility{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.RBDAbility), err
}

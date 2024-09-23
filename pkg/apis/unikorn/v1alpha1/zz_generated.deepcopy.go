//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2024 the Unikorn Authors.

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	unikornv1alpha1 "github.com/unikorn-cloud/core/pkg/apis/unikorn/v1alpha1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationBundleAutoUpgradeSpec) DeepCopyInto(out *ApplicationBundleAutoUpgradeSpec) {
	*out = *in
	if in.WeekDay != nil {
		in, out := &in.WeekDay, &out.WeekDay
		*out = new(ApplicationBundleAutoUpgradeWeekDaySpec)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationBundleAutoUpgradeSpec.
func (in *ApplicationBundleAutoUpgradeSpec) DeepCopy() *ApplicationBundleAutoUpgradeSpec {
	if in == nil {
		return nil
	}
	out := new(ApplicationBundleAutoUpgradeSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationBundleAutoUpgradeWeekDaySpec) DeepCopyInto(out *ApplicationBundleAutoUpgradeWeekDaySpec) {
	*out = *in
	if in.Sunday != nil {
		in, out := &in.Sunday, &out.Sunday
		*out = new(ApplicationBundleAutoUpgradeWindowSpec)
		**out = **in
	}
	if in.Monday != nil {
		in, out := &in.Monday, &out.Monday
		*out = new(ApplicationBundleAutoUpgradeWindowSpec)
		**out = **in
	}
	if in.Tuesday != nil {
		in, out := &in.Tuesday, &out.Tuesday
		*out = new(ApplicationBundleAutoUpgradeWindowSpec)
		**out = **in
	}
	if in.Wednesday != nil {
		in, out := &in.Wednesday, &out.Wednesday
		*out = new(ApplicationBundleAutoUpgradeWindowSpec)
		**out = **in
	}
	if in.Thursday != nil {
		in, out := &in.Thursday, &out.Thursday
		*out = new(ApplicationBundleAutoUpgradeWindowSpec)
		**out = **in
	}
	if in.Friday != nil {
		in, out := &in.Friday, &out.Friday
		*out = new(ApplicationBundleAutoUpgradeWindowSpec)
		**out = **in
	}
	if in.Saturday != nil {
		in, out := &in.Saturday, &out.Saturday
		*out = new(ApplicationBundleAutoUpgradeWindowSpec)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationBundleAutoUpgradeWeekDaySpec.
func (in *ApplicationBundleAutoUpgradeWeekDaySpec) DeepCopy() *ApplicationBundleAutoUpgradeWeekDaySpec {
	if in == nil {
		return nil
	}
	out := new(ApplicationBundleAutoUpgradeWeekDaySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationBundleAutoUpgradeWindowSpec) DeepCopyInto(out *ApplicationBundleAutoUpgradeWindowSpec) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationBundleAutoUpgradeWindowSpec.
func (in *ApplicationBundleAutoUpgradeWindowSpec) DeepCopy() *ApplicationBundleAutoUpgradeWindowSpec {
	if in == nil {
		return nil
	}
	out := new(ApplicationBundleAutoUpgradeWindowSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationBundleSpec) DeepCopyInto(out *ApplicationBundleSpec) {
	*out = *in
	if in.Version != nil {
		in, out := &in.Version, &out.Version
		*out = new(string)
		**out = **in
	}
	if in.Preview != nil {
		in, out := &in.Preview, &out.Preview
		*out = new(bool)
		**out = **in
	}
	if in.EndOfLife != nil {
		in, out := &in.EndOfLife, &out.EndOfLife
		*out = (*in).DeepCopy()
	}
	if in.Applications != nil {
		in, out := &in.Applications, &out.Applications
		*out = make([]ApplicationNamedReference, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationBundleSpec.
func (in *ApplicationBundleSpec) DeepCopy() *ApplicationBundleSpec {
	if in == nil {
		return nil
	}
	out := new(ApplicationBundleSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationBundleStatus) DeepCopyInto(out *ApplicationBundleStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationBundleStatus.
func (in *ApplicationBundleStatus) DeepCopy() *ApplicationBundleStatus {
	if in == nil {
		return nil
	}
	out := new(ApplicationBundleStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationNamedReference) DeepCopyInto(out *ApplicationNamedReference) {
	*out = *in
	if in.Name != nil {
		in, out := &in.Name, &out.Name
		*out = new(string)
		**out = **in
	}
	if in.Reference != nil {
		in, out := &in.Reference, &out.Reference
		*out = new(unikornv1alpha1.ApplicationReference)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationNamedReference.
func (in *ApplicationNamedReference) DeepCopy() *ApplicationNamedReference {
	if in == nil {
		return nil
	}
	out := new(ApplicationNamedReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterManager) DeepCopyInto(out *ClusterManager) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterManager.
func (in *ClusterManager) DeepCopy() *ClusterManager {
	if in == nil {
		return nil
	}
	out := new(ClusterManager)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterManager) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterManagerApplicationBundle) DeepCopyInto(out *ClusterManagerApplicationBundle) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterManagerApplicationBundle.
func (in *ClusterManagerApplicationBundle) DeepCopy() *ClusterManagerApplicationBundle {
	if in == nil {
		return nil
	}
	out := new(ClusterManagerApplicationBundle)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterManagerApplicationBundle) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterManagerApplicationBundleList) DeepCopyInto(out *ClusterManagerApplicationBundleList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterManagerApplicationBundle, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterManagerApplicationBundleList.
func (in *ClusterManagerApplicationBundleList) DeepCopy() *ClusterManagerApplicationBundleList {
	if in == nil {
		return nil
	}
	out := new(ClusterManagerApplicationBundleList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterManagerApplicationBundleList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterManagerList) DeepCopyInto(out *ClusterManagerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterManager, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterManagerList.
func (in *ClusterManagerList) DeepCopy() *ClusterManagerList {
	if in == nil {
		return nil
	}
	out := new(ClusterManagerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterManagerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterManagerSpec) DeepCopyInto(out *ClusterManagerSpec) {
	*out = *in
	if in.ApplicationBundle != nil {
		in, out := &in.ApplicationBundle, &out.ApplicationBundle
		*out = new(string)
		**out = **in
	}
	if in.ApplicationBundleAutoUpgrade != nil {
		in, out := &in.ApplicationBundleAutoUpgrade, &out.ApplicationBundleAutoUpgrade
		*out = new(ApplicationBundleAutoUpgradeSpec)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterManagerSpec.
func (in *ClusterManagerSpec) DeepCopy() *ClusterManagerSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterManagerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterManagerStatus) DeepCopyInto(out *ClusterManagerStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]unikornv1alpha1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterManagerStatus.
func (in *ClusterManagerStatus) DeepCopy() *ClusterManagerStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterManagerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *File) DeepCopyInto(out *File) {
	*out = *in
	if in.Path != nil {
		in, out := &in.Path, &out.Path
		*out = new(string)
		**out = **in
	}
	if in.Content != nil {
		in, out := &in.Content, &out.Content
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new File.
func (in *File) DeepCopy() *File {
	if in == nil {
		return nil
	}
	out := new(File)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesCluster) DeepCopyInto(out *KubernetesCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesCluster.
func (in *KubernetesCluster) DeepCopy() *KubernetesCluster {
	if in == nil {
		return nil
	}
	out := new(KubernetesCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KubernetesCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesClusterAPISpec) DeepCopyInto(out *KubernetesClusterAPISpec) {
	*out = *in
	if in.SubjectAlternativeNames != nil {
		in, out := &in.SubjectAlternativeNames, &out.SubjectAlternativeNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.AllowedPrefixes != nil {
		in, out := &in.AllowedPrefixes, &out.AllowedPrefixes
		*out = make([]unikornv1alpha1.IPv4Prefix, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesClusterAPISpec.
func (in *KubernetesClusterAPISpec) DeepCopy() *KubernetesClusterAPISpec {
	if in == nil {
		return nil
	}
	out := new(KubernetesClusterAPISpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesClusterApplicationBundle) DeepCopyInto(out *KubernetesClusterApplicationBundle) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesClusterApplicationBundle.
func (in *KubernetesClusterApplicationBundle) DeepCopy() *KubernetesClusterApplicationBundle {
	if in == nil {
		return nil
	}
	out := new(KubernetesClusterApplicationBundle)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KubernetesClusterApplicationBundle) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesClusterApplicationBundleList) DeepCopyInto(out *KubernetesClusterApplicationBundleList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]KubernetesClusterApplicationBundle, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesClusterApplicationBundleList.
func (in *KubernetesClusterApplicationBundleList) DeepCopy() *KubernetesClusterApplicationBundleList {
	if in == nil {
		return nil
	}
	out := new(KubernetesClusterApplicationBundleList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KubernetesClusterApplicationBundleList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesClusterControlPlaneSpec) DeepCopyInto(out *KubernetesClusterControlPlaneSpec) {
	*out = *in
	in.MachineGeneric.DeepCopyInto(&out.MachineGeneric)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesClusterControlPlaneSpec.
func (in *KubernetesClusterControlPlaneSpec) DeepCopy() *KubernetesClusterControlPlaneSpec {
	if in == nil {
		return nil
	}
	out := new(KubernetesClusterControlPlaneSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesClusterFeaturesSpec) DeepCopyInto(out *KubernetesClusterFeaturesSpec) {
	*out = *in
	if in.Autoscaling != nil {
		in, out := &in.Autoscaling, &out.Autoscaling
		*out = new(bool)
		**out = **in
	}
	if in.NvidiaOperator != nil {
		in, out := &in.NvidiaOperator, &out.NvidiaOperator
		*out = new(bool)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesClusterFeaturesSpec.
func (in *KubernetesClusterFeaturesSpec) DeepCopy() *KubernetesClusterFeaturesSpec {
	if in == nil {
		return nil
	}
	out := new(KubernetesClusterFeaturesSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesClusterList) DeepCopyInto(out *KubernetesClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]KubernetesCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesClusterList.
func (in *KubernetesClusterList) DeepCopy() *KubernetesClusterList {
	if in == nil {
		return nil
	}
	out := new(KubernetesClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KubernetesClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesClusterNetworkSpec) DeepCopyInto(out *KubernetesClusterNetworkSpec) {
	*out = *in
	in.NetworkGeneric.DeepCopyInto(&out.NetworkGeneric)
	if in.PodNetwork != nil {
		in, out := &in.PodNetwork, &out.PodNetwork
		*out = (*in).DeepCopy()
	}
	if in.ServiceNetwork != nil {
		in, out := &in.ServiceNetwork, &out.ServiceNetwork
		*out = (*in).DeepCopy()
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesClusterNetworkSpec.
func (in *KubernetesClusterNetworkSpec) DeepCopy() *KubernetesClusterNetworkSpec {
	if in == nil {
		return nil
	}
	out := new(KubernetesClusterNetworkSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesClusterSpec) DeepCopyInto(out *KubernetesClusterSpec) {
	*out = *in
	if in.Version != nil {
		in, out := &in.Version, &out.Version
		*out = new(unikornv1alpha1.SemanticVersion)
		**out = **in
	}
	if in.Network != nil {
		in, out := &in.Network, &out.Network
		*out = new(KubernetesClusterNetworkSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.API != nil {
		in, out := &in.API, &out.API
		*out = new(KubernetesClusterAPISpec)
		(*in).DeepCopyInto(*out)
	}
	if in.ControlPlane != nil {
		in, out := &in.ControlPlane, &out.ControlPlane
		*out = new(KubernetesClusterControlPlaneSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.WorkloadPools != nil {
		in, out := &in.WorkloadPools, &out.WorkloadPools
		*out = new(KubernetesClusterWorkloadPoolsSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Features != nil {
		in, out := &in.Features, &out.Features
		*out = new(KubernetesClusterFeaturesSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.ApplicationBundle != nil {
		in, out := &in.ApplicationBundle, &out.ApplicationBundle
		*out = new(string)
		**out = **in
	}
	if in.ApplicationBundleAutoUpgrade != nil {
		in, out := &in.ApplicationBundleAutoUpgrade, &out.ApplicationBundleAutoUpgrade
		*out = new(ApplicationBundleAutoUpgradeSpec)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesClusterSpec.
func (in *KubernetesClusterSpec) DeepCopy() *KubernetesClusterSpec {
	if in == nil {
		return nil
	}
	out := new(KubernetesClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesClusterStatus) DeepCopyInto(out *KubernetesClusterStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]unikornv1alpha1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesClusterStatus.
func (in *KubernetesClusterStatus) DeepCopy() *KubernetesClusterStatus {
	if in == nil {
		return nil
	}
	out := new(KubernetesClusterStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesClusterWorkloadPoolsPoolSpec) DeepCopyInto(out *KubernetesClusterWorkloadPoolsPoolSpec) {
	*out = *in
	in.KubernetesWorkloadPoolSpec.DeepCopyInto(&out.KubernetesWorkloadPoolSpec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesClusterWorkloadPoolsPoolSpec.
func (in *KubernetesClusterWorkloadPoolsPoolSpec) DeepCopy() *KubernetesClusterWorkloadPoolsPoolSpec {
	if in == nil {
		return nil
	}
	out := new(KubernetesClusterWorkloadPoolsPoolSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesClusterWorkloadPoolsSpec) DeepCopyInto(out *KubernetesClusterWorkloadPoolsSpec) {
	*out = *in
	if in.Pools != nil {
		in, out := &in.Pools, &out.Pools
		*out = make([]KubernetesClusterWorkloadPoolsPoolSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesClusterWorkloadPoolsSpec.
func (in *KubernetesClusterWorkloadPoolsSpec) DeepCopy() *KubernetesClusterWorkloadPoolsSpec {
	if in == nil {
		return nil
	}
	out := new(KubernetesClusterWorkloadPoolsSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesWorkloadPoolSpec) DeepCopyInto(out *KubernetesWorkloadPoolSpec) {
	*out = *in
	in.MachineGeneric.DeepCopyInto(&out.MachineGeneric)
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Files != nil {
		in, out := &in.Files, &out.Files
		*out = make([]File, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Autoscaling != nil {
		in, out := &in.Autoscaling, &out.Autoscaling
		*out = new(MachineGenericAutoscaling)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesWorkloadPoolSpec.
func (in *KubernetesWorkloadPoolSpec) DeepCopy() *KubernetesWorkloadPoolSpec {
	if in == nil {
		return nil
	}
	out := new(KubernetesWorkloadPoolSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineGeneric) DeepCopyInto(out *MachineGeneric) {
	*out = *in
	in.MachineGeneric.DeepCopyInto(&out.MachineGeneric)
	if in.FlavorName != nil {
		in, out := &in.FlavorName, &out.FlavorName
		*out = new(string)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineGeneric.
func (in *MachineGeneric) DeepCopy() *MachineGeneric {
	if in == nil {
		return nil
	}
	out := new(MachineGeneric)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineGenericAutoscaling) DeepCopyInto(out *MachineGenericAutoscaling) {
	*out = *in
	if in.MinimumReplicas != nil {
		in, out := &in.MinimumReplicas, &out.MinimumReplicas
		*out = new(int)
		**out = **in
	}
	if in.MaximumReplicas != nil {
		in, out := &in.MaximumReplicas, &out.MaximumReplicas
		*out = new(int)
		**out = **in
	}
	if in.Scheduler != nil {
		in, out := &in.Scheduler, &out.Scheduler
		*out = new(MachineGenericAutoscalingScheduler)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineGenericAutoscaling.
func (in *MachineGenericAutoscaling) DeepCopy() *MachineGenericAutoscaling {
	if in == nil {
		return nil
	}
	out := new(MachineGenericAutoscaling)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineGenericAutoscalingScheduler) DeepCopyInto(out *MachineGenericAutoscalingScheduler) {
	*out = *in
	if in.CPU != nil {
		in, out := &in.CPU, &out.CPU
		*out = new(int)
		**out = **in
	}
	if in.Memory != nil {
		in, out := &in.Memory, &out.Memory
		x := (*in).DeepCopy()
		*out = &x
	}
	if in.GPU != nil {
		in, out := &in.GPU, &out.GPU
		*out = new(MachineGenericAutoscalingSchedulerGPU)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineGenericAutoscalingScheduler.
func (in *MachineGenericAutoscalingScheduler) DeepCopy() *MachineGenericAutoscalingScheduler {
	if in == nil {
		return nil
	}
	out := new(MachineGenericAutoscalingScheduler)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineGenericAutoscalingSchedulerGPU) DeepCopyInto(out *MachineGenericAutoscalingSchedulerGPU) {
	*out = *in
	if in.Type != nil {
		in, out := &in.Type, &out.Type
		*out = new(string)
		**out = **in
	}
	if in.Count != nil {
		in, out := &in.Count, &out.Count
		*out = new(int)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineGenericAutoscalingSchedulerGPU.
func (in *MachineGenericAutoscalingSchedulerGPU) DeepCopy() *MachineGenericAutoscalingSchedulerGPU {
	if in == nil {
		return nil
	}
	out := new(MachineGenericAutoscalingSchedulerGPU)
	in.DeepCopyInto(out)
	return out
}

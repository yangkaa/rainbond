// RAINBOND, Application Management Platform
// Copyright (C) 2014-2017 Goodrain Co., Ltd.

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

package volume

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
)

// NFSVolume NFS volume struct
type NFSVolume struct {
	Base
}

// CreateVolume nfs create volume
func (n *NFSVolume) CreateVolume(define *Define) error {
	NFSName := fmt.Sprintf("nfs-%d", n.svm.ID)
	volumes := corev1.Volume{
		Name: NFSName,
		VolumeSource: corev1.VolumeSource{
			NFS: &corev1.NFSVolumeSource{
				Server: n.svm.NFSServer,
				Path:   n.svm.NFSPath,
			},
		},
	}
	define.volumes = append(define.volumes, volumes)
	volumeMounts := corev1.VolumeMount{
		Name:      NFSName,
		MountPath: n.svm.VolumePath,
	}
	define.volumeMounts = append(define.volumeMounts, volumeMounts)
	return nil
}

// CreateDependVolume nfs create depend volume
func (n *NFSVolume) CreateDependVolume(define *Define) error {
	return nil
}

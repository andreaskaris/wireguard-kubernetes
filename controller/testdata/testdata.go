package testdata

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var masterNode0 = &corev1.Node{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Node",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "master-0",
		Labels: map[string]string{
			"node-role.kubernetes.io/master": "",
			"kubernetes.io/hostname":         "master-0",
		},
		Annotations: map[string]string{
			"wireguard.kubernetes.io/publickey": "bDOPiAaYvtq1y+7+u75t1QYhogY4cuLo02jPhjNM+FA=",
		},
	},
	Spec: corev1.NodeSpec{
		Taints: []corev1.Taint{
			corev1.Taint{
				Key:    "node-role.kubernetes.io/master",
				Effect: corev1.TaintEffectNoSchedule,
			},
		},
		PodCIDR: "10.245.0.0/24",
		PodCIDRs: []string{
			"10.245.0.0/24",
		},
	},
	Status: corev1.NodeStatus{
		Addresses: []corev1.NodeAddress{
			corev1.NodeAddress{
				Type:    corev1.NodeInternalIP,
				Address: "172.18.0.100",
			},
		},
	},
}

var masterNode1 = &corev1.Node{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Node",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "master-1",
		Labels: map[string]string{
			"node-role.kubernetes.io/master": "",
			"kubernetes.io/hostname":         "master-1",
		},
		Annotations: map[string]string{
			"wireguard.kubernetes.io/publickey": "jQyD90Rm1xTj5YkYTrgUTc2AVgHqUbwFpvVUSCUV/Ao=",
		},
	},
	Spec: corev1.NodeSpec{
		Taints: []corev1.Taint{
			corev1.Taint{
				Key:    "node-role.kubernetes.io/master",
				Effect: corev1.TaintEffectNoSchedule,
			},
		},
		PodCIDR: "10.245.1.0/24",
		PodCIDRs: []string{
			"10.245.1.0/24",
		},
	},
	Status: corev1.NodeStatus{
		Addresses: []corev1.NodeAddress{
			corev1.NodeAddress{
				Type:    corev1.NodeInternalIP,
				Address: "172.18.0.101",
			},
		},
	},
}

var masterNode2 = &corev1.Node{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Node",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "master-2",
		Labels: map[string]string{
			"node-role.kubernetes.io/master": "",
			"kubernetes.io/hostname":         "master-2",
		},
		Annotations: map[string]string{
			"wireguard.kubernetes.io/publickey": "UOoRnP0Tn/MTFOo2ciOGQcudIqsHcN5UVevvmZ2k7TI=",
		},
	},
	Spec: corev1.NodeSpec{
		Taints: []corev1.Taint{
			corev1.Taint{
				Key:    "node-role.kubernetes.io/master",
				Effect: corev1.TaintEffectNoSchedule,
			},
		},
		PodCIDR: "10.245.2.0/24",
		PodCIDRs: []string{
			"10.245.2.0/24",
		},
	},
	Status: corev1.NodeStatus{
		Addresses: []corev1.NodeAddress{
			corev1.NodeAddress{
				Type:    corev1.NodeInternalIP,
				Address: "172.18.0.102",
			},
		},
	},
}

var WorkerNode0 = &corev1.Node{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Node",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "worker-0",
		Labels: map[string]string{
			"node-role.kubernetes.io/worker": "",
			"kubernetes.io/hostname":         "worker-0",
		},
		Annotations: map[string]string{
			"wireguard.kubernetes.io/publickey": "qP+1Sstf6Y0MYBeUtJjWthBMfx8uG1hmK4mz9hOQjGI=",
		},
	},
	Spec: corev1.NodeSpec{
		PodCIDR: "10.245.3.0/24",
		PodCIDRs: []string{
			"10.245.3.0/24",
		},
	},
	Status: corev1.NodeStatus{
		Addresses: []corev1.NodeAddress{
			corev1.NodeAddress{
				Type:    corev1.NodeInternalIP,
				Address: "172.18.0.103",
			},
		},
	},
}

var WorkerNode1 = &corev1.Node{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Node",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "worker-1",
		Labels: map[string]string{
			"node-role.kubernetes.io/worker": "",
			"kubernetes.io/hostname":         "worker-1",
		},
		Annotations: map[string]string{
			"wireguard.kubernetes.io/publickey": "KmmEwqKHPxZIE2T1dRW51nj4V45W/0eIDibwEinlmQo=",
		},
	},
	Spec: corev1.NodeSpec{
		PodCIDR: "10.245.4.0/24",
		PodCIDRs: []string{
			"10.245.4.0/24",
		},
	},
	Status: corev1.NodeStatus{
		Addresses: []corev1.NodeAddress{
			corev1.NodeAddress{
				Type:    corev1.NodeInternalIP,
				Address: "172.18.0.104",
			},
		},
	},
}

var WorkerNode2 = &corev1.Node{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Node",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "worker-2",
		Labels: map[string]string{
			"node-role.kubernetes.io/worker": "",
			"kubernetes.io/hostname":         "worker-2",
		},
		Annotations: map[string]string{
			"wireguard.kubernetes.io/publickey": "dsrxnDAs1KBvvuGuTxi4cr2i/csK+fFCzaq4mX6Mfj0=",
		},
	},
	Spec: corev1.NodeSpec{
		PodCIDR: "10.245.5.0/24",
		PodCIDRs: []string{
			"10.245.5.0/24",
		},
	},
	Status: corev1.NodeStatus{
		Addresses: []corev1.NodeAddress{
			corev1.NodeAddress{
				Type:    corev1.NodeInternalIP,
				Address: "172.18.0.105",
			},
		},
	},
}

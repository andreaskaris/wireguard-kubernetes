package testdata

import (
	"os/exec"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var MasterNode0 = &corev1.Node{
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
			//			"wireguard.kubernetes.io/tunnel-ip": "100.64.0.101",
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
			"2000::3/64",
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

var MasterNode1 = &corev1.Node{
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
			//			"wireguard.kubernetes.io/tunnel-ip": "100.64.0.102",
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

var MasterNode2 = &corev1.Node{
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
			//			"wireguard.kubernetes.io/tunnel-ip": "100.64.0.103",
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
			//			"wireguard.kubernetes.io/tunnel-ip": "100.64.0.104",
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
			//			"wireguard.kubernetes.io/tunnel-ip": "100.64.0.105",
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
			//			"wireguard.kubernetes.io/tunnel-ip": "100.64.0.106",
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

var WorkerNodeLocal = &corev1.Node{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Node",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "worker-local",
		Labels: map[string]string{
			"node-role.kubernetes.io/worker": "",
			"kubernetes.io/hostname":         "worker-local",
		},
		Annotations: map[string]string{
			"wireguard.kubernetes.io/publickey": "qP+1Sstf6Y0MYBeUtJjWthBMfx8uG1hmK4mz9hOQjGI=",
			//			"wireguard.kubernetes.io/tunnel-ip": "100.64.0.104",
		},
	},
	Spec: corev1.NodeSpec{
		PodCIDR: "10.245.6.0/24",
		PodCIDRs: []string{
			"10.245.6.0/24",
		},
	},
	Status: corev1.NodeStatus{
		Addresses: []corev1.NodeAddress{
			corev1.NodeAddress{
				Type: corev1.NodeInternalIP,
				Address: func() string {
					cmd := "ip -4 -o a ls dev $(ip r | awk '/default/ {print $5}') | awk '{print $4}' | awk -F '/' '{print $1}'"
					out, err := exec.Command("/bin/bash", "-c", cmd).Output()
					if err != nil {
						return "172.18.0.1"
					}
					return strings.Trim(string(out), "\n")
				}(),
			},
		},
	},
}

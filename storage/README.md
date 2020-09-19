# How to create NFS StorageClass

Kubernetes cluster supports persistent volume by using PersistenVolume (PV) and Persistent Volume Claim (PVC) 
which are cluster storage resource. We need to define PV with underlying actual storage then define PVC that 
use it, then bind PVC to an application.

Kubernetes also support Storage Class in which instead of define PV manually, we can set storage class to application 
PVC as an annotation. Then, cluster will create a PV that match the storage class for us automatically.

**Example:**
```
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: pvc-test
  annotations:
    volume.beta.kubernetes.io/storage-class: "ssdnfs"
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 1Gi
```
Each cloud provider like AWS or Google has its own support for storage class that bind the actual storage with their 
existing cloud storage.
Since our kubernetes cluster is on a VMs or bare metal, we don't have this capability out of the box.
In this post, I will guide you to setup storage class based on NFS using a program named "nfs-client-provisioner".

**Prerequisite:**

1. A kubernetes cluster on VMs or bare metal with RBAC enabled
2. A NFS server

We will create a storage class name `ssdnfs` as a default storage class.
Let's assume that we have NFS server on IP 192.168.1.119 and export path /export/k8sdynamic.


**1. Create storage class**

Run command below to define our storage class to the cluster.
```
echo 'apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: ssdnfs
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: k8s/nfs' | kubectl apply -f -
```
**2. Create service account and permission**

The provisioner needs permission to monitor and create PV for us so we need a service account with appropriate 
permission.

Create service account.
```
echo "apiVersion: v1
kind: ServiceAccount
metadata:
  name: nfs-client-provisioner" | kubectl apply -f -
```

Define cluster role.
```
echo 'kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: nfs-client-provisioner-runner
rules:
  - apiGroups: [""]
    resources: ["persistentvolumes"]
    verbs: ["get", "list", "watch", "create", "delete"]
  - apiGroups: [""]
    resources: ["persistentvolumeclaims"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["storageclasses"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["list", "watch", "create", "update", "patch"]
  - apiGroups: [""]
    resources: ["endpoints"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]' | kubectl apply -f -
```
Bind cluster to service account.
```
echo 'kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: run-nfs-client-provisioner
subjects:
  - kind: ServiceAccount
    name: nfs-client-provisioner
    namespace: default
roleRef:
  kind: ClusterRole
  name: nfs-client-provisioner-runner
  apiGroup: rbac.authorization.k8s.io' | kubectl apply -f -
```
**3. Deploy provisioner as pod**

Run command below to deploy nfs-client-provisioner application to the cluster.
```
echo 'kind: Deployment
apiVersion: apps/v1
metadata:
  name: nfs-client-provisioner
spec:
  selector:
      matchLabels:
        app: nfs-client-provisioner
  replicas: 1
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app: nfs-client-provisioner
    spec:
      serviceAccount: nfs-client-provisioner
      containers:
        - name: nfs-client-provisioner
          image: quay.io/external_storage/nfs-client-provisioner:v3.1.0-k8s1.11
          volumeMounts:
            - name: nfs-client-root
              mountPath: /persistentvolumes
          env:
            - name: PROVISIONER_NAME
              value: k8s/nfs
            - name: NFS_SERVER
              value: 192.168.1.119 # nodes will need nfs-common to access nfs protocol
            - name: NFS_PATH
              value: /export/k8sdynamic
      volumes:
        - name: nfs-client-root
          nfs:
            server: 192.168.1.119
            path: /export/k8sdynamic' | kubectl apply -f -
```
Run kubectl get deployment nfs-client-provisioner and wait until you see

**4. Test our storage class**

We will create PVC that use our "ssdnfs" storage class and run a pod that use this PVC.

Create PVC.
```
echo 'kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: test-claim
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 1Mi
  storageClassName: ssdnfs' | kubectl apply -f -
```
Create test pod.
```
echo 'kind: Pod
apiVersion: v1
metadata:
  name: test-pod
spec:
  containers:
  - name: test-pod
    image: gcr.io/google_containers/busybox:1.24
    command:
      - "/bin/sh"
    args:
      - "-c"
      - "touch /mnt/SUCCESS && exit 0 || exit 1"
    volumeMounts:
      - name: nfs-pvc
        mountPath: "/mnt"
  restartPolicy: "Never"
  volumes:
    - name: nfs-pvc
      persistentVolumeClaim:
        claimName: test-claim' | kubectl apply -f -
```
Run kubectl get pod test-pod and if you see status "completed", it means our storage class works!



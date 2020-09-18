# k8sMLer-client-go
Based on kubernetes/client-go API to talk with Kubernetes
打造出一套基于API的自定义集群操作工具

**INSTALL:**

```
git clone https://github.com/ReyRen/k8sMLer-client-go.git
```
to your `GOPATH` src dir

```
go build .
./k8sMLer-client-go -h
```

**USAGE:**
```
-o={create、delete、list}[list]
	表示操作的类型: 创建、删除和展示 
-n={string}[""]
	表示指定已有的命名空间，这个参数是必须存在的
-r={pod,service}[""]
	表示指定资源类型，这个参数是必须存在的
-k={string}[""]
	表示指定创建资源的命名，在创建时可以不指定(推荐指定)
	但在删除(非标签化删除)、展示(非标签化展示)下必须指定
-l={string}[""]
	表示指定标签，用于对多个资源进行管理(删除和展示)
	当使用标签化删除pods的时候，-k不需要指定
-g={int}[0]
	表示使用的gpu资源个数，这个个数时单台所能支持的卡数
```
NOTE: {}表示可选参数列表，[]表示默认参数

**PROGRESS:**

2020.09.18:
完成单个service和pod的创建、删除、查看功能以及根据标签删除pods, 根据标签查看service和pods

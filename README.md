# k8sMLer-client-go
**实现功能:**
1. 使用kubernetes/client-go进行kubernetes集群资源控制
2. 使用websocket作为与前端的交互
3. 支持多任务和多用户的同时训练和页面刷新后，实时训练日志的正确重定向展示（支持动态刷新ws）

**设计架构:**
![hub](https://github.com/ReyRen/k8sMLer-client-go/blob/master/Hub.jpg)

**部署:**
###### go version >= 1.14 (go module require go version >= 1.12)
设置代理，否则google官方源被墙掉了
```
export GOPROXY=https://goproxy.io
```
进入`GOPATH` src dir下

```
git clone https://github.com/ReyRen/k8sMLer-client-go.git
```
然后执行编译或编译加运行
```
make (build)
或者
make run
```

**使用注意:**

这里创建的storageclass是"web-nfs"的名字，请参考[StorageClass-creation](https://github.com/ReyRen/k8sMLer-client-go/blob/master/storage/README.md)提前创建
并且更改代码中相应的命名

因为涉及到传参到集群pod中进行分布式训练，所以会有两个脚本在`scripts/`目录中

请体检创建好名字为web的namespace, 或者更改common.go中相应的宏

更改common.go中关于server IP的宏

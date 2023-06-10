`Alma` 版本 `8.6`\
\
`etcd` 版本 `3.5.5`

# 部署 `etcd` 集群

## 1.初始化系统

> \[!NOTE]\
> 每台 `etcd` 服务器都要设置

### 1-1.设置 `hostname`

> \[!NOTE]\
> 设置后重启生效

```
hostnamectl set-hostname etcd-01
hostnamectl set-hostname etcd-02
hostnamectl set-hostname etcd-03

```

### 1-2.设置 `/etc/hosts`

```
cat >> /etc/hosts <<EOF
192.168.192.128  etcd-01
192.168.192.131  etcd-02
192.168.192.132  etcd-03
EOF

```

### 1-3.禁用 `selinux`

临时禁用和永久禁用

```
setenforce 0 && \
sed -ri '/^SELINUX/s/(SELINUX=)(.*)/\1disabled/' /etc/selinux/config

```

### 1-4.禁用 `firewalld` 和 `iptables`

> \[!NOTE]\
> 如果 `iptables` 没有安装则不用管

```
systemctl stop firewalld && systemctl disable firewalld && \
systemctl stop iptables && systemctl disable iptables

```

### 1-5.创建目录

```
mkdir -p /etc/etcd/ssl/ && \
mkdir -p /ssl/

```

### 1-6.设置接口版本变量

> \[!NOTE]\
> 新版本的 `etcd` 使用的是 `v3` 版本的接口，有提供向下兼容 `v2` 版本的接口。\
> \
> 如果需要，设置成使用 `v3` 版本的。这里使用的是 `v3` 版本。\
> \
> 每个节点都需要设置。

```
cat >> /etc/profile <<EOF
export ETCDETC_API=3
EOF
source /etc/profile

```

## 2.创建 `ssl` 证书

> \[!NOTE]\
> 使用的是 `cloudflare` 提供的 `cfssl` 证书创建工具。\
> \
> \
> 在 `etcd` 集群中需要创建 `server` , `peer` 两种证书：\\
>
> *   `server` 证书用于 `etcd` 开启 `ssl` 认证
> *   `peer` 证书用于集群内节点双向通信
>
> 创建证书可以在任何地方创建，创建后上传到 `etcd` 服务器就行。

### 2-1.下载 `cfssl`

```
cd /home && \

wget https://pkg.cfssl.org/R1.2/cfssl_linux-amd64 && \

wget https://pkg.cfssl.org/R1.2/cfssljson_linux-amd64 && \

chmod +x cfssl_linux-amd64 cfssljson_linux-amd64 && \

mv cfssl_linux-amd64 /usr/local/bin/cfssl && \

mv cfssljson_linux-amd64 /usr/local/bin/cfssljson

```

### 2-2.创建证书

#### 2-2-1.创建 `ca` 配置文件

```
cat > /ssl/ca-config.json <<EOF
{
    "signing": {
        "default": {
            "expiry": "43800h"
        },
        "profiles": {
            "server": {
                "expiry": "43800h",
                "usages": [
                    "signing",
                    "key encipherment",
                    "server auth"
                ]
            },
            "client": {
                "expiry": "43800h",
                "usages": [
                    "signing",
                    "key encipherment",
                    "client auth"
                ]
            },
            "peer": {
                "expiry": "43800h",
                "usages": [
                    "signing",
                    "key encipherment",
                    "server auth",
                    "client auth"
                ]
            }
        }
    }
}
EOF

```

#### 2-2-2.创建 `ca` 证书

> \[!NOTE]\
> 该操作将生成 `ca-key.pem` , `ca.csr` , `ca.pem` 三个证书。\
> \
> \
> `*.csr` 证书在整个过程中将不会用到。要确保 `ca.key` 证书的安全。\
> \
> \
> 根证书用 `4096` 位保证安全。

```
cat > /ssl/etcd-ca-csr.json <<EOF
{
    "key": {
        "algo": "rsa",
        "size": 4096
    },
    "names": [
        {
            "C": "CN",
            "L": "Beijing",
            "ST": "Beijing",
            "O": "k8s",
            "OU": "System"
        }
    ]
}
EOF
cd /ssl/ && \
cfssl gencert -initca etcd-ca-csr.json | \
cfssljson -bare etcd-ca - && \
ls etcd-ca* | grep etcd-ca

```

#### 2-2-3.创建 `server` 证书

> \[!NOTE]\
> 这里需要把 `etcd` 集群的 `ip` 地址都写进去。

```
cat > /ssl/etcd-server-csr.json <<EOF
{
    "CN": "etcd-server",
    "hosts": [
        "127.0.0.1",
        "192.168.192.128",
        "192.168.192.131",
        "192.168.192.132"
    ],
    "key": {
        "algo": "rsa",
        "size": 2048
    },
    "names": [
        {
            "C": "CN",
            "L": "Beijing",
            "O": "etcd-cluster",
            "OU": "System",
            "ST": "Beijing"
        }
    ]
}
EOF
cd /ssl/ && \
cfssl gencert \
-ca=etcd-ca.pem \
-ca-key=etcd-ca-key.pem \
-config=ca-config.json \
-profile=server etcd-server-csr.json | \
cfssljson -bare etcd-server && \
ls etcd-server* | grep etcd-server

```

#### 2-2-4.创建 `peer` 双向对等证书

> \[!NOTE]\
> 对于 `etcd` 集群，每个节点既是作为 `server` 又是作为 `client` 端，所以需要生成 `peer` 对等证书。\
> \
> 同样需要把所有节点的 `ip` 地址都写进去。

```
cat > /ssl/etcd-peer-csr.json <<EOF
{
    "CN": "etcd-peer",
    "hosts": [
        "127.0.0.1",
        "192.168.192.128",
        "192.168.192.131",
        "192.168.192.132"
    ],
    "key": {
        "algo": "rsa",
        "size": 2048
    },
    "names": [
        {
            "C": "CN",
            "L": "Beijing",
            "O": "etcd-cluster",
            "OU": "System",
            "ST": "Beijing"
        }
    ]
}
EOF
cd /ssl/ && \
cfssl gencert \
-ca=etcd-ca.pem -ca-key=etcd-ca-key.pem \
-config=ca-config.json \
-profile=peer etcd-peer-csr.json | \
cfssljson -bare etcd-peer && \
ls etcd-peer* | grep etcd-peer

```

### 2-3.分发证书

#### 2-3-1.分发 `ca` 证书

```
scp /ssl/etcd-ca.pem root@192.168.192.128:/etc/etcd/ssl/ && \
scp /ssl/etcd-ca.pem root@192.168.192.131:/etc/etcd/ssl/ && \
scp /ssl/etcd-ca.pem root@192.168.192.132:/etc/etcd/ssl/

```

#### 2-3-2.分发 `server` 证书

```
scp /ssl/etcd-server*.pem root@192.168.192.128:/etc/etcd/ssl/ && \

scp /ssl/etcd-server*.pem root@192.168.192.131:/etc/etcd/ssl/ && \

scp /ssl/etcd-server*.pem root@192.168.192.132:/etc/etcd/ssl/

```

#### 2-3-3.分发 `peer` 证书

```
scp /ssl/etcd-peer*.pem root@192.168.192.128:/etc/etcd/ssl/ && \

scp /ssl/etcd-peer*.pem root@192.168.192.131:/etc/etcd/ssl/ && \

scp /ssl/etcd-peer*.pem root@192.168.192.132:/etc/etcd/ssl/

```

## 3.部署 `etcd`

### 3-1.创建目录

> \[!NOTE]\
> 每台服务器都要创建。\
> \
> 工作目录为：`/var/lib/etcd`\
> \
> 数据存放目录：`/data/etcd`\
> \
> 配置文件地址：`/etc/etcd/etcd.conf`\
> \
> 集群部署方式为：`static`。其他部署方式还有：`etcd discovery` 以及 `DNS discovery`

```
mkdir /var/lib/etcd/ && \
mkdir -p /data/etcd/ && \
chmod -R 777 /data/etcd/

```

### 3-2.下载 `etcd`

下载对应硬件架构的 `etcd` 二进制文件

```
cd /home && \
wget https://github.com/etcd-io/etcd/releases/download/v3.5.5/etcd-v3.5.5-linux-amd64.tar.gz

```

解压

```
tar -zxvf etcd-v3.5.5-linux-amd64.tar.gz

```

复制命令到 `/usr/local/bin/`

```
cd /home/etcd-v3.5.5-linux-amd64/ && \
cp etcd etcdctl etcdutl /usr/local/bin/

```

分发二进制文件到每台服务器

```
scp etcd etcdctl etcdutl root@192.168.192.128:/usr/local/bin/ && \

scp etcd etcdctl etcdutl root@192.168.192.131:/usr/local/bin/ && \

scp etcd etcdctl etcdutl root@192.168.192.132:/usr/local/bin/

```

### 3-3.创建 `etcd.conf`

> \[!NOTE]\
> 每台服务器都要创建。\
> \
> `ip` 地址改为自己对应的。

#### 3-3-1.在 `etcd-01` 服务器创建

```
cat > /etc/etcd/etcd.conf <<EOF
ETCD_NAME=etcd-01
ETCD_DATA_DIR="/data/etcd/"

ETCD_LISTEN_CLIENT_URLS="https://192.168.192.128:2379,https://127.0.0.1:2379"

ETCD_LISTEN_PEER_URLS="https://192.168.192.128:2380"

ETCD_INITIAL_ADVERTISE_PEER_URLS="https://192.168.192.128:2380"

ETCD_INITIAL_CLUSTER="etcd-01=https://192.168.192.128:2380,etcd-02=https://192.168.192.131:2380,etcd-03=https://192.168.192.132:2380"

ETCD_INITIAL_CLUSTER_STATE="new"
ETCD_INITIAL_CLUSTER_TOKEN="etcd-cluster"

ETCD_ADVERTISE_CLIENT_URLS="https://192.168.192.128:2379"

ETCD_CLIENT_CERT_AUTH="true"
ETCD_TRUSTED_CA_FILE="/etc/etcd/ssl/etcd-ca.pem"
ETCD_CERT_FILE="/etc/etcd/ssl/etcd-server.pem"
ETCD_KEY_FILE="/etc/etcd/ssl/etcd-server-key.pem"

ETCD_PEER_CLIENT_CERT_AUTH="true"
ETCD_PEER_TRUSTED_CA_FILE="/etc/etcd/ssl/etcd-ca.pem"
ETCD_PEER_CERT_FILE="/etc/etcd/ssl/etcd-peer.pem"
ETCD_PEER_KEY_FILE="/etc/etcd/ssl/etcd-peer-key.pem"
EOF

```

#### 3-3-2.在 `etcd-02` 服务器创建

```
cat > /etc/etcd/etcd.conf <<EOF
ETCD_NAME=etcd-02
ETCD_DATA_DIR="/data/etcd/"

ETCD_LISTEN_CLIENT_URLS="https://192.168.192.131:2379,https://127.0.0.1:2379"

ETCD_LISTEN_PEER_URLS="https://192.168.192.131:2380"

ETCD_INITIAL_ADVERTISE_PEER_URLS="https://192.168.192.131:2380"

ETCD_INITIAL_CLUSTER="etcd-01=https://192.168.192.128:2380,etcd-02=https://192.168.192.131:2380,etcd-03=https://192.168.192.132:2380"

ETCD_INITIAL_CLUSTER_STATE="new"
ETCD_INITIAL_CLUSTER_TOKEN="etcd-cluster"

ETCD_ADVERTISE_CLIENT_URLS="https://192.168.192.131:2379"

ETCD_CLIENT_CERT_AUTH="true"
ETCD_TRUSTED_CA_FILE="/etc/etcd/ssl/etcd-ca.pem"
ETCD_CERT_FILE="/etc/etcd/ssl/etcd-server.pem"
ETCD_KEY_FILE="/etc/etcd/ssl/etcd-server-key.pem"

ETCD_PEER_CLIENT_CERT_AUTH="true"
ETCD_PEER_TRUSTED_CA_FILE="/etc/etcd/ssl/etcd-ca.pem"
ETCD_PEER_CERT_FILE="/etc/etcd/ssl/etcd-peer.pem"
ETCD_PEER_KEY_FILE="/etc/etcd/ssl/etcd-peer-key.pem"
EOF

```

#### 3-3-4.在 `etcd-03` 服务器创建

```
cat > /etc/etcd/etcd.conf <<EOF
ETCD_NAME=etcd-03
ETCD_DATA_DIR="/data/etcd/"

ETCD_LISTEN_CLIENT_URLS="https://192.168.192.132:2379,https://127.0.0.1:2379"

ETCD_LISTEN_PEER_URLS="https://192.168.192.132:2380"

ETCD_INITIAL_ADVERTISE_PEER_URLS="https://192.168.192.132:2380"

ETCD_INITIAL_CLUSTER="etcd-01=https://192.168.192.128:2380,etcd-02=https://192.168.192.131:2380,etcd-03=https://192.168.192.132:2380"

ETCD_INITIAL_CLUSTER_STATE="new"
ETCD_INITIAL_CLUSTER_TOKEN="etcd-cluster"

ETCD_ADVERTISE_CLIENT_URLS="https://192.168.192.132:2379"

ETCD_CLIENT_CERT_AUTH="true"
ETCD_TRUSTED_CA_FILE="/etc/etcd/ssl/etcd-ca.pem"
ETCD_CERT_FILE="/etc/etcd/ssl/etcd-server.pem"
ETCD_KEY_FILE="/etc/etcd/ssl/etcd-server-key.pem"

ETCD_PEER_CLIENT_CERT_AUTH="true"
ETCD_PEER_TRUSTED_CA_FILE="/etc/etcd/ssl/etcd-ca.pem"
ETCD_PEER_CERT_FILE="/etc/etcd/ssl/etcd-peer.pem"
ETCD_PEER_KEY_FILE="/etc/etcd/ssl/etcd-peer-key.pem"
EOF

```

### 3-4.创建启动服务

> \[!NOTE]\
> 每台服务器都创建，启动项都一样。

```
cat > /usr/lib/systemd/system/etcd.service <<EOF
[Unit]
Description=Etcd Server
After=network.target

[Service]
Type=simple
WorkingDirectory=/var/lib/etcd/
EnvironmentFile=-/etc/etcd/etcd.conf
# set GOMAXPROCS to number of processors
ExecStart=/bin/bash -c "GOMAXPROCS=$(nproc) /usr/local/bin/etcd"
Type=notify

[Install]
WantedBy=multi-user.target
EOF

```

### 3-5.启动服务

> \[!NOTE]\
> 每台服务器都执行。\
> \
> 由于是部署集群而不是单节点部署，所以要打开三个 `ssh` 标签都连接到服务器并且尽可能同时执行启动。\
> \
> 因为每个服务都会在一定时间内监听其他服务的状态把集群节点加入进来。

```
systemctl start etcd && \
systemctl enable etcd

```

### 3-6.检查状态

检查健康状态

```
etcdctl --cacert=/etc/etcd/ssl/etcd-ca.pem \
--cert=/etc/etcd/ssl/etcd-peer.pem \
--key=/etc/etcd/ssl/etcd-peer-key.pem \
--endpoints="https://192.168.192.128:2379,https://192.168.192.131:2379,https://192.168.192.132:2379" \
endpoint health --write-out="table"

```

*显示。可以看到 `HEALTH` 状态为 `true`*

```
+-----------------------------+--------+-------------+-------+
|          ENDPOINT           | HEALTH |    TOOK     | ERROR |
+-----------------------------+--------+-------------+-------+
| https://192.168.192.128:2379 |   true | 20.329616ms |       |
| https://192.168.192.131:2379 |   true | 24.033545ms |       |
| https://192.168.192.132:2379 |   true | 23.918639ms |       |
+-----------------------------+--------+-------------+-------+

```

查看成员列表

```
etcdctl --cacert=/etc/etcd/ssl/etcd-ca.pem \
--cert=/etc/etcd/ssl/etcd-peer.pem \
--key=/etc/etcd/ssl/etcd-peer-key.pem \
--endpoints="https://192.168.192.128:2379,https://192.168.192.131:2379,https://192.168.192.132:2379" \
member list --write-out="table"

```

*显示*

```
+------------------+---------+---------+-----------------------------+-----------------------------+------------+
|        ID        | STATUS  |  NAME   |         PEER ADDRS          |        CLIENT ADDRS         | IS LEARNER |
+------------------+---------+---------+-----------------------------+-----------------------------+------------+
| 1cdccba2db807b88 | started | etcd-01 | https://192.168.192.128:2380 | https://192.168.192.128:2379 |      false |
| 58badf9cfae9256f | started | etcd-03 | https://192.168.192.132:2380 | https://192.168.192.132:2379 |      false |
| 7cf1d6781c958345 | started | etcd-02 | https://192.168.192.131:2380 | https://192.168.192.131:2379 |      false |
+------------------+---------+---------+-----------------------------+-----------------------------+------------+

```

### 3-7.其他操作

停止服务

```
systemctl stop etcd && \
systemctl disable etcd && \
systemctl daemon-reload

```

如果重启报错需要删除旧的数据

```
rm -rvf /data/etcd/* && rm -rvf /var/lib/etcd/*

```

删除 `ssl` 证书

```
rm -rvf /etc/etcd/ssl/*

```

删除配置文件

```
rm -rvf /etc/etcd/etcd.conf

```

***

## 至此。 `etcd` 集群部署成功。

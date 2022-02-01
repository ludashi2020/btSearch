# btSearch  一个用go语言实现的磁力搜索网站  

## 1.页面展示

![image](https://raw.githubusercontent.com/Bmixo/btSearch/master/example/index_old.png)
![image](https://raw.githubusercontent.com/Bmixo/btSearch/master/example/index.PNG)
![image](https://raw.githubusercontent.com/Bmixo/btSearch/master/example/detail.PNG)
## 2.程序架构
名称   |  用途
|------------:|-----------
server |  收集torrent数据
worker | 收集Hash信息
web    |  数据展示
Tool   | 工具

![image](https://raw.githubusercontent.com/Bmixo/btSearch/master/example/framework.png)
## 注意:
1.项目使用了reuseport系统特性来监听端口，请保持新的linux 内核版本（4.9以上）

## 3.安装
1. 在安装环境前您需要配置golang环境  [Go](https://golang.org/) 下载地址
2. 安装python和mongodb
```
sudo apt install mongodb
sudo apt install python3 python3-pip
pip3 install elastic-doc-manager[elastic5]
```
3. 服务端安装
```
go get github.com/Bmixo/btSearch/server

```
4. worker安装
```
go get github.com/Bmixo/btSearch/worker
```

5. Elasticsearch dockerp配置
```
docker run --restart=always -p 9200:9200 -p 9300:9300 --name=tmp docker.elastic.co/elasticsearch/elasticsearch:5.6.0
```

安装analysis-ik分词器
```
docker ps
docker exec -it $dockid /bin/bash
./bin/elasticsearch-plugin install https://github.com/medcl/elasticsearch-analysis-ik/releases/download/v5.6.0/elasticsearch-analysis-ik-5.6.0.zip

```
配置elasticsearch的数据存放目录
```
mkdir -p /data/docker
docker cp $dockid:/usr/share/elasticsearch/ /data/docker
chmod 777 -R /data/
```
运行Elasticsearch docker服务
```
docker run --restart=always -p 9200:9200 -p 9300:9300 --name=es \
-e ES_JAVA_OPTS="-Xms1024m -Xmx1024m" \
-v /data/docker/elasticsearch/data:/usr/share/elasticsearch/data \
-v /data/docker/elasticsearch/logs:/usr/share/elasticsearch/logs \
-v /data/docker/elasticsearch/config:/usr/share/elasticsearch/config \
-v /data/docker/elasticsearch/plugins:/usr/share/elasticsearch/plugins \
-v /etc/localtime:/etc/localtime \
-v /etc/timezone:/etc/timezone \
docker.elastic.co/elasticsearch/elasticsearch:5.6.0
```

配置正确的话执行下面命令可以看到当前Elasticsearch的运行状态
```
curl --user elastic:changeme -XGET 'http://127.0.0.1:9200/_cat/health'
``` 
配置Elasticsearch默认分词器
```
curl --user elastic:changeme -XPUT http://localhost:9200/bavbt -H 'Content-Type: application/json'
curl --user elastic:changeme -XPOST 'localhost:9200/bavbt/_close'
curl --user elastic:changeme -XPUT localhost:9200/bavbt/_settings?pretty -d '{
"index":{
"analysis" : {
            "analyzer" : {
                "default" : {
                    "type" : "ik_max_word"
                }
            },
			"search_analyzer" : {
                "default" : {
                    "type" : "ik_max_word"
                }
            }
        }
    }
}'
curl --user elastic:changeme -XPOST 'localhost:9200/bavbt/_open'
```



## 4.运行

首先需要配置您的conf文件
```
[mongodb]
musername=
mpassword=
database=bavbt
collection=torrent
addr=127.0.0.1:27017
[elasticsearch]
url=http://127.0.0.1:9200/bavbt/torrent/
eusername=elastic
epassword=changeme
[webServer]
webServerAddr=127.0.0.1:7878
```


服务端运行
```
go run github.com/Bmixo/btSearch/server
```
worker运行
```
go run github.com/Bmixo/btSearch/worker
```


## 5.TODO
- [ ] 后台数据展示
- [ ] 打包docker镜像
- [ ] 提供k8s高可用部署方案（mongodb sharding + 无状态均衡负载master  + etcd）
- [ ] gin迁移iris








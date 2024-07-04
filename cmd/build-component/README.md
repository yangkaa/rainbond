## 代码说明

该代码用于生成 Rainbond 应用模版的 json 文件，通过替换数据库 json 文件，可以快速构造一个包含大量组件的应用模版。

具体操作如下：

1. 克隆该代码，修改 componentNum 变量，指定组件数量（如不指定，默认数量为100），然后执行 `go run cmd/build-component/generate_app_model.go` 生成模版 json 文件
2. 在 Rainbond 平台，创建一个应用包含一个nginx，将其发布成应用模版
3. 连接 console 数据库，找到 `rainbond_center_app_version` 表，找到第二步创建的应用模版对应的版本，将生成的 json 文件替换到 `rainbond_center_app_version` 表的 `app_template` 字段中
4. 回到 Rainbond 平台，找到发布的应用模版，点击"安装"，即可安装该模版所包含的所有组件
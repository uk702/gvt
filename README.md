安装：  
go get -u github.com/uk702/gvt  

20170314:
1）支持镜像 url，比如 golang.org/x/...，这个地址已经被 google 废弃，通常不能直接下载，而是需要到 github.com/golang/ 里下载，因此，需要在 mirrorUrls 中增加如下的一行：
golang.org/x/ github.com/golang/

然后将这个文件保存在个人目录下（如 C:\Users\Administrator），这样，gvt 下载第三方依赖时，遇到  golang.org/x/...，就会到 github.com/golang/... 进行下载。

2）新增 gvt init 命令，这个命令会扫描所有需要的第三方依赖，然后下载到 vendor 目录下，以 beego 为例：

1. 下载 beego
cd /d f:\workspace
git clone https://github.com/astaxie/beego.git beego/src/github.com/astaxie/beego

2. 设置 GOPATH
set GOPATH=f:/workspace/beego

3. （首次运行时）将 mirrorUrls 这个文件拷贝到个人目录下

4. 下载相关依赖
cd beego/src/github.com/astaxie/beego
gvt init （如果因为网络的原因可能下载不完整，这个命令可以执行多遍）

5. 编译
确认所有的依赖都下载成功后，执行 go install ./... 进行编译，应该编译成功，生成的 exe 保存在 f:/workspace/beego/bin 目录下。


20161231:  
使用：  
1、下载第三方代码（连同它的依赖）：  
gvt fetch github.com/spf13/hugo  

相关的代码将下载到 vendor 目录下。
下载过程中，manifest 文件将记录成功下载的第三方依赖，failFetchUrls 则记录下所有失败的下载。  
  
2、下载所有失败的依赖  
如是由于网络不稳定的造成的下载失败（比如下载到一半就超时），那么，可以通过如下命令重新下载失败的那些依赖：  
gvt fetch fix  
  
这个命令将读取 failFetchUrls 这个文件，并下载其中记录的所有失败的依赖。  
  
3、主要用法
1） gvt fetch github.com/spf13/hugo  
下载所有源代码，但不包括测试文件（*_test.go）和其它不相关的文件（比如数据文件、配置文件，可能所有非 *.go 的文件都被归为“不相关文件”）  
  
2) gvt fetch -a github.com/spf13/hugo  
下载所有文件，这当然就包括测试文件了。  
如果你发现 gvt fetch xxx 下载“不完全”时，可改用 gvt fetch -a xxx 这个命令再下载  
  
3) gvt fetch -a -v github.com/spf13/hugo  
下载所有文件，下载时显示 git clone 的进度，如  
remote: Counting objects: 156, done.  
remote: Compressing objects: 100% (125/125), done.  
remote: Total 156 (delta 1), reused 87 (delta 0), pack-reused 0Receiving objects  
Receiving objects: 100% (156/156), 164.00 KiB | 49.00 KiB/s  
Receiving objects: 100% (156/156), 179.92 KiB | 49.00 KiB/s, done.  
  

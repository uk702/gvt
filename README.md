安装：  
go get -u github.com/uk702/gvt  
  
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
  

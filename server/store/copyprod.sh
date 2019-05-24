#/bin/bash
tool=`dirname "$0"`/tool.go

go run $tool -emul=false config-get | go run $tool -emul=true config-set
go run $tool -emul=false conf-get | go run $tool -emul=true conf-set

#mkdir build
#~/go/bin/xgo --targets=linux/amd64 --dest=build github.com/adriamb/gometh-server
chmod +x build/gometh-server-linux-amd64
cp -R ../../../../../gometh-contracts/build/contracts build

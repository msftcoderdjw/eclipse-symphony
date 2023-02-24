# Symphony CLI (maestro)

## Download and install


### Linux/Mac
```bash
wget -q https://raw.githubusercontent.com/Haishi2016/Vault818/master/cli/install/install.sh -O - | /bin/bash
```
### Windows
```cmd
powershell -Command "iwr -useb https://raw.githubusercontent.com/Haishi2016/Vault818/master/cli/install/install.ps1 | iex"
```

## Install Symphony
Install all prerequisites and Symphony, including:
* Docker
* Kubernetes (Kind)
* Kubectl
* Helm 3
* Symphony API
* Symphony Portal (with ```--with-portal``` switch)
```bash
./maestro up
```

## Check prerequisites
Check all Symphony depedencies
```bash
./maestro check
```

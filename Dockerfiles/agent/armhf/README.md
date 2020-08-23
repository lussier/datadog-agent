# Build instructions

## Tested on

- Hardware: Raspberry Pi 4
- OS: Raspberry Pi OS Lite 2020-05-27, based on Debian buster
- Docker Engine: 19.03.12
- libseccomp2: 2.4.3-1+b1

## Prerequisites

- [Install the latest Docker Engine](https://docs.docker.com/engine/install/debian/).
- Manually install a newer version of libseccomp because the one that's currently in buster repository has [this issue](https://github.com/moby/moby/issues/40734), which blocks syscalls required to run some of the image build steps. 
  ``` 
  wget https://packages.debian.org/sid/armhf/libseccomp2/download
  sudo dpkg -i <PACKAGE>.pkg
  ```  

## Build Debian package

- [Build the Datadog Agent Debian package](https://www.fonz.net/blog/archives/2020/06/19/datadog-v7-on-raspberry-pi2/) using pre-built build images:
  ```
  git clone https://github.com/DataDog/datadog-agent.git
  cd datadog-agent
  docker run -v "$PWD:/go/src/github.com/DataDog/datadog-agent" -v "/tmp/omnibus:/omnibus" --workdir=/go/src/github.com/DataDog/datadog-agent irabinovitch/datadog-agent-buildimages-armhf:latest inv -e agent.omnibus-build --base-dir=/omnibus --gem-path=/gem
  ```
  Once done with building, go look for the deb package under `/tmp/omnibus`.  

## Build Docker image

- Put the `datadog-agent_*armhf.deb` package under `./Dockerfiles/agent/`.
- Run Docker build:
    ``` 
    cd ./Dockerfiles/agent/
    docker build -f armhf/Dockerfile .
    ``` 

## Pre-built images

I pushed the images I built in Docker Hub if you'd like to use them directly: [lussier/datadog-agent](https://hub.docker.com/repository/docker/lussier/datadog-agent).

## Dockerfile

The armhf Dockerfile I added is an integral copy of the arm64 with two changes:

- Find and replace arm64 with armhf
- Added `-r` to `xargs` here because find would output nothing and that would exit the build with an error: 
  ```
  RUN find /etc -type d,f -perm -o+w -print0 | xargs -0 -r chmod g-w,o-w
  ```
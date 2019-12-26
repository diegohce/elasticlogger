# Plugin parameters
PLUGIN_NAME=elasticlogger
PLUGIN_TAG=latest

all: clean build build-image create-plugin

clean:
	@echo "Removing plugin and build directory"
	rm -f ./es-log-driver
	rm -rf ./plugin-build

build:
	go build -o ./es-log-driver .

build-image:
	@echo "docker build: rootfs image with the plugin"
	docker build -f Dockerfile.build -t ${PLUGIN_NAME}:rootfsimg .
	@echo "### create rootfs directory in ./plugin-build/rootfs"
	mkdir -p ./plugin-build/rootfs
	docker create --name ${PLUGIN_NAME}-rootfs ${PLUGIN_NAME}:rootfsimg
	docker export ${PLUGIN_NAME}-rootfs | tar -x -C ./plugin-build/rootfs
	mkdir -p ./plugin-build/rootfs/run/docker/plugins
	@echo "### copy config.json to ./plugin-build/"
	cp config.json ./plugin-build/
	docker rm -vf ${PLUGIN_NAME}-rootfs
	docker rmi ${PLUGIN_NAME}:rootfsimg

create-plugin:
	@echo "### remove existing plugin ${PLUGIN_NAME}:${PLUGIN_TAG} if exists"
	docker plugin rm -f ${PLUGIN_NAME}:${PLUGIN_TAG} || true
	@echo "### create new plugin ${PLUGIN_NAME}:${PLUGIN_TAG} from ./plugin-build"
	docker plugin create ${PLUGIN_NAME}:${PLUGIN_TAG} ./plugin-build
	@echo "### plugin ${PLUGIN_NAME}:${PLUGIN_TAG} is disabled"
	@echo "### configure ${PLUGIN_NAME}:${PLUGIN_TAG} first"
	@echo "### then run 'docker plugin enable ${PLUGIN_NAME}:${PLUGIN_TAG}'"
#	@echo "### enable plugin ${PLUGIN_NAME}:${PLUGIN_TAG}"
#	docker plugin enable ${PLUGIN_NAME}:${PLUGIN_TAG}


ZONE := us-east1-b

# заменить на нормальное понятное после тестов
VM_NAME := codenire-vm
PROJ := codiewio
NETWORK := default
TAG_NAME := codenire
RESERVED_IP := codenireip


# Docker environment for the sandbox server itself (containing docker CLI, etc), running
# in a privileged container.
docker:
	docker build -f Dockerfile --tag=codenire/codenire-sandbox ..

# dockergvisor builds the golang/playground-sandbox-gvisor docker
# image, which is the environment that the untrusted programs run in
# (a busybox:glibc world with this directory's sandbox binary which
# runs in --mode=contained)
dockergvisor:
	docker build -f Dockerfile.gvisor --tag=codenire/codenire-sandbox-gvisor ..
	docker tag codenire/codenire-sandbox-gvisor registry.digitalocean.com/codiew/codenire-sandbox-gvisor:latest

push: docker dockergvisor
	docker push registry.digitalocean.com/codiew/codenire-sandbox:latest

# runlocal runs the sandbox server locally, for use with the frontend
# parent directory's "test_nacl" or "test_gvisor" test targets.
runlocal: docker dockergvisor
	docker network create sandnet || true
	docker kill sandbox_dev || true
	docker run --name=sandbox_dev --rm --network=sandnet -ti -p 127.0.0.1:8080:80/tcp -v /var/run/docker.sock:/var/run/docker.sock golang/playground-sandbox:latest --dev

konlet.yaml.expanded: konlet.yaml
	sed "s/PROJECT_NAME/$(PROJ)/" konlet.yaml > konlet.yaml.expanded

# create_test_vm creates a test VM for interactive debugging.
create_vm: konlet.yaml.expanded
	gcloud --project=$(PROJ) compute instances create $(VM_NAME) \
	--zone $(ZONE) \
	--network $(NETWORK) \
	--no-address \
	--tags $(TAG_NAME) \
	--image-project cos-cloud \
	--image cos-stable-76-12239-60-0 \
	--metadata-from-file gce-container-declaration=konlet.yaml.expanded,user-data=cloud-init.yaml


#-address=$(RESERVED_IP) \

# delete_test_vm deletes the test VM from create_vm.
delete_test_vm:
	gcloud --project=$(PROJ) compute instances delete $(VM_NAME) --quiet --zone $(ZONE)

create_firewall:
	gcloud compute firewall-rules create allow-$(VM_NAME) \
      --allow tcp:80,tcp:22 \
      --network default \
      --source-ranges 0.0.0.0/0 \
      --target-tags $(TAG_NAME)

delete-firewall:
	gcloud compute firewall-rules delete allow-$(VM_NAME) --quiet

# ssh connects to the create_test_vm VM. It must be run from the same network.
ssh:
	gcloud --project=$(PROJ) compute ssh $(VM_NAME) --internal-ip --zone $(ZONE) --ssh-key-file=~/.ssh/id_ed25519


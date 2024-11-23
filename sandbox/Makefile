ZONE := us-east1-b

# заменить на нормальное понятное после тестов
TEST_VM := codenire-test-vm
PROJ := codenire
NETWORK := default
TAG_NAME := codenire
RESERVED_IP := codenircom


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
	gcloud --project=$(PROJ) compute instances create $(TEST_VM) \
	--zone $(ZONE) \
	--network $(NETWORK) \
	--address=$(RESERVED_IP) \
	--tags $(TAG_NAME) \
	--image-project cos-cloud \
	--image cos-stable-76-12239-60-0 \
	--metadata-from-file gce-container-declaration=konlet.yaml.expanded,user-data=cloud-init.yaml

# delete_test_vm deletes the test VM from create_vm.
delete_test_vm:
	gcloud --project=$(PROJ) compute instances delete $(TEST_VM) --quiet --zone $(ZONE)

create-firewall:
	gcloud compute firewall-rules create allow-$(TEST_VM) \
      --allow tcp:80,tcp:443,tcp:22 \
      --network default \
      --source-ranges 0.0.0.0/0 \
      --target-tags $(TAG_NAME)

delete-firewall:
	gcloud compute firewall-rules delete allow-$(TEST_VM) --quiet

# ssh connects to the create_test_vm VM. It must be run from the same network.
ssh:
	gcloud --project=$(PROJ) compute ssh $(TEST_VM) --internal-ip --zone $(ZONE) --ssh-key-file=~/.ssh/id_ed25519


m:
	gcloud auth configure-docker \
		us-east1-docker.pkg.dev


delete:
	curl -X DELETE -H "Authorization: Bearer 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" \
	  "https://ghcr.io/v2/codenire/codenire-sandbox/manifests/sha256:e8548e2b01cd0ab275b68f08f9d28203c56389769f7d9917214e31ac3d0006c6"

images:
	curl -H "Authorization: Bearer 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" \
         -H "Accept: application/vnd.github.v3+json" \
         "https://api.github.com/orgs/codenire/packages/container"
	curl -H "Authorization: Bearer YOUR_PERSONAL_ACCESS_TOKEN" \
         -H "Accept: application/vnd.github.v3+json" \
         "https://api.github.com/users/YOUR_USERNAME/packages"
# Сodenire Playground

Open-source online code execution system featuring
a playground and sandbox.
Built on Docker images with robust isolation provided by [Google gVisor](https://github.com/google/gvisor).

The system is easily extensible with additional technologies and languages.

Inspired by:
- Judje0 Playground: https://github.com/judge0/judge0
- Google Playground: https://github.com/golang/playground



<a href="https://codiew.io" target="_blank">
  <img width="1242" alt="Screenshot 2025-02-04 at 01 04 36" src="https://github.com/user-attachments/assets/dc79a2c8-b095-4987-909e-239f0d0afc74" />
</a>


*Playground demonstration in the [codiew.io](https://codiew.io) service.*

Special thanks to:

<a href="https://codiew.io" target="_blank">
<img width="130" alt="Screenshot 2025-01-30 at 17 19 20" src="https://github.com/user-attachments/assets/db4350d0-31a2-46cf-9e69-ef24b0075650" />
</a>


# 🌟Features

- Multi-Language Support: Run Python, Go, and Node.js code in isolated Docker containers.
- Multi-Files Support: Run code that consists of multiple files
- Easy extensible: You can create your own build to run code via API or by loading a folder with configurations at startup.
- (in working) Dependency Management: Automatic handling of project dependencies (pip, go mod, npm)
- Flexible Execution: Custom entrypoints for both single-file code and full projects
- Scalable System: Expandable via a load balancer between a playground and a sandbox.


# 🐙Infrastructure Schema

![Image alt](docs/docs/general_schema.png)


# Sandbox Provision Templates Schema

![Image alt](docs/docs/sandbox_schema.png)

**[!] The ability to register Docker images via API is not yet implemented and will be available in the near future!**

Out of the box (in development),
Dockerfiles and configurations for various languages can be found in /sandbox/dockerfiles

# Usage Playground

```
# Input request result:

POST https://codenire.com/run
Content-Type: application/json

{
  "templateId": "golang_1_23",
  "files": {
    "main.go": "package main\n\nimport (\n\t\"flag\"\n\t\"fmt\"\n)\n\nfunc main() {\n\t// Process command-line arguments\n\tname := flag.String(\"name\", \"default\", \"User name\")\n\tflag.Parse()\n\n\t// Read data from stdin\n\tvar input string\n\t_, err := fmt.Scan(&input)\n\tif err != nil {\n\t\tfmt.Println(\"Error reading from stdin:\", err)\n\t\treturn\n\t}\n\n\t// Print arguments and stdin data\n\tfmt.Printf(\"Hello, %s!\\n\", *name)\n\tfmt.Printf(\"Stdin data: %s\\n\", input)\n}\n"
  },
  "args": "--name \"Mark\"",
  "stdin": "100.00"
}



# Output result:

{
  "Events": [
    {
      "Kind": "stdout",
      "Message": "Hello, Mark!\nStdin data: 100.00\n"
    }
  ]
}
```

# Run/Set Up
You can Run Playground local (or on MacOS/Ubuntu) via Docker Compose.

**[!] If you start on MacOS you can't start with gVisor Environment**

```yaml
services:
  playground:
    container_name: play_dev
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8081:8081"
    volumes:
      # You can set up your path with go-plugin 
      - ./var/plugin/plugin:/plugin
      # You can set up your path with plugin scripts (see docs/docker-compose dir with examples)
      - ./var/plugin/hooks-dir:hooks-dir
    restart: always
    networks:
      - "sandnet"
    entrypoint: [
      "/playground",
      "--backend-url", "http://sandbox_dev/run",
      "--port", "8081",
      "--hooks-plugins", "/plugin",
      #      "--hooks-dir", "/hooks-dir",
    ]

  sandbox:
    container_name: sandbox_dev
    build:
      context: ./sandbox
      dockerfile: Dockerfile
    ports:
      - "8082:80"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      # You can set up your path with configs 
      - ./var/dockerfiles:/dockerfiles
    networks:
      - sandnet
    restart: always
    entrypoint: [
      "/usr/local/bin/sandbox",
      "--dockerFilesPath", "/dockerfiles",
      "--replicaContainerCnt", "1",
      "--port", "80",
    ]

networks:
  sandnet:
    name: codenire


```

# Deploy

- Docker compose (see [/docs/docker-compose](https://github.com/codiewio/codenire/tree/main/docs/docker-compose) dir — without external gVisor Runtime)
- [Digital Ocean Terraform](docs/digitalocean/README.md) with load balancing and multi-sandbox cluster


# Lifecycle Request Hooks

TODO:: add description

# Roadmap
- [x] Add Multifiles/singe scripts
- [x] Add gVisor Isolation
- [x] Add Hooks to catch/override some request (for auth, for handle code in external system)
- [x] Add Multi actions in once container (different runs in one docker img, for example multi version of c++ in cpp container)
- [ ] Add Metrics
- [ ] Add Tests
- [ ] Add golinter
- [ ] Add debug info in API req/resp
- [x] Change container replica strategy
- [ ] Add Statefull sandbox
- [x] Compile with open network
- [ ] Add WS messaging

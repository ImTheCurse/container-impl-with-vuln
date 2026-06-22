# Containment Demo Server

This service is for demonstrating the container limits in this repository without
adding a real application vulnerability.

## Routes

- `GET /health`: basic liveness check
- `POST /upload`: uploads a file into the container's local `uploads/` directory
- `GET /uploads`: lists uploaded files
- `GET /demo/containment`: runs controlled probes and reports what the process can
  and cannot do inside the container

## What the demo shows

- The process can write inside its own jailed filesystem
- The process sees an isolated `/proc` and hostname
- Mount attempts should fail when seccomp/capability restrictions are effective

## Important limitation

The Go runtime currently creates `UTS`, `PID`, and `mount` namespaces, but not a
network namespace. That means outbound networking may still be possible from the
contained process unless you add network isolation separately.

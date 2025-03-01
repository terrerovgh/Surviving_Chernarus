# MDS Grid Renderer Docker Container

This project contains a terminal-based graphical renderer that displays an animated grid using the tcell library. The application has been containerized using Docker for easy deployment and execution.

## Prerequisites

- Docker
- Docker Compose (optional, but recommended)

## Building and Running the Container

### Using Docker Compose (Recommended)

The easiest way to build and run the application is using Docker Compose:

```bash
# Build and start the container with default TTY device (/dev/tty1)
docker-compose up -d

# Start with a specific TTY device
TTY_DEVICE=/dev/tty2 docker-compose up -d

# Run in the current terminal (recommended for interactive use)
docker-compose run --rm mdsrenderer

# View the logs
docker-compose logs -f

# Stop the container
docker-compose down
```

### Using Docker Directly

Alternatively, you can use Docker commands directly:

```bash
# Build the image
docker build -t mdsrenderer .

# Run the container with default TTY device
docker run -it --name mdsrenderer mdsrenderer

# Run the container with a specific TTY device
docker run -it --name mdsrenderer -e TTY_DEVICE=/dev/tty2 --device /dev/tty2:/dev/tty2 mdsrenderer

# Run in the current terminal (recommended for interactive use)
docker run -it --rm mdsrenderer

# Stop and remove the container
docker stop mdsrenderer
docker rm mdsrenderer
```

## Notes

- The application requires terminal capabilities to display properly, which is why we use the `-it` flags and set `tty: true` in the Docker Compose file.
- The environment variable `TERM=xterm-256color` is set to ensure proper color support.
- The application will display a grid with animated traces and occasional scrambling effects.
- When running with `docker-compose run` or `docker run -it --rm`, the output will be displayed in your current terminal, providing the best interactive experience.

## Troubleshooting

If you encounter display issues:

1. Ensure your terminal supports the required features (colors, Unicode characters)
2. Try connecting to the container directly: `docker exec -it mdsrenderer sh`
3. Check the container logs: `docker logs mdsrenderer`
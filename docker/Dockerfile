# Use official OpenJDK 21 slim image
FROM eclipse-temurin:21-jre-jammy

# Install curl, jq and cleanup in same layer
RUN apt-get update && \
    apt-get install -y curl jq && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Create non-root user for security \
RUN useradd -m -u 1000 -s /bin/bash minecraft

# Set working directory (This is where PVC will mount)
WORKDIR /data

# Env variables
ENV VERSION=1.21.11 \
  GAME_MODE=survival \
  DIFFICULTY=normal \
  MAX_PLAYERS=20 \
  VIEW_DISTANCE=10 \
  PVP=true \
  ENABLE_COMMAND_BLOCK=false \
  SPAWN_PROTECTION=16 \
  MOTD="Test Server" \
  JAVA_MEMORY=768M

# Copy startup script
COPY start.sh /start.sh
RUN chmod +x /start.sh

# Change ownership of /data to minecraft user
RUN chown -R minecraft:minecraft /data

# Switch to non-root user
USER minecraft

# Expose Minecraft port
EXPOSE 25565

# Health check: Verify server is responding on port 25565
HEALTHCHECK --interval=30s --timeout=10s --start-period=120s --retries=3 \
    CMD timeout 5 bash -c '</dev/tcp/localhost/25565' || exit 1

# Run startup script
ENTRYPOINT ["/start.sh"]
#!/bin/bash

cd /data || exit

# Download Paper server if not present
if [ ! -f server.jar ]; then
  echo "Downloading Paper server version 1.21.11"

  PROJECT="paper"
  USER_AGENT="kubecraft/1.0.0 (baig.hasan@outlook.com)"

  # First check if the requested version has a stable build
  BUILDS_RESPONSE=$(curl -s -H "User-Agent: $USER_AGENT" https://fill.papermc.io/v3/projects/${PROJECT}/versions/${VERSION}/builds)

  # Check if the API returned an error
  if echo "$BUILDS_RESPONSE" | jq -e '.ok == false' > /dev/null 2>&1; then
    ERROR_MSG=$(echo "$BUILDS_RESPONSE" | jq -r '.message // "Unknown error"')
    echo "Error: $ERROR_MSG"
    exit 1
  fi

  # Try to get a stable build URL for the requested version
  PAPERMC_URL=$(echo "$BUILDS_RESPONSE" | jq -r 'first(.[] | select(.channel == "STABLE") | .downloads."server:default".url) // "null"')
  FOUND_VERSION="$MINECRAFT_VERSION"

  # If no stable build for requested version, find the latest version with a stable build
  if [ "$PAPERMC_URL" == "null" ]; then
    echo "No stable build for version $MINECRAFT_VERSION, searching for latest version with stable build..."
    exit 1
  fi
    # Download the latest Paper version
    curl -o server.jar $PAPERMC_URL
    echo "Download completed (version: $FOUND_VERSION)"
fi

# Accept EULA automatically
if [ ! -f eula.txt ]; then
  echo "eula=true" > eula.txt
fi

# Generate server.properties from environment variables
# Generate server.properties from environment variables
#cat > server.properties << EOF
## Kubecraft Auto-Generated Configuration
#server-port=25565
#gamemode=${GAME_MODE}
#difficulty=${DIFFICULTY}
#max-players=${MAX_PLAYERS}
#view-distance=${VIEW_DISTANCE}
#pvp=${PVP}
#enable-command-block=${ENABLE_COMMAND_BLOCK}
#spawn-protection=${SPAWN_PROTECTION}
#motd=${MOTD}
#online-mode=true
#white-list=false
#spawn-monsters=true
#spawn-animals=true
#spawn-npcs=true
#EOF

echo "Starting Minecraft server..."
echo "Memory: ${JAVA_MEMORY}"
echo "Version: ${VERSION}"

# Start server with optimized JVM flags
exec java -Xms${JAVA_MEMORY} -Xmx${JAVA_MEMORY} \
    -XX:+UseG1GC \
    -XX:+ParallelRefProcEnabled \
    -XX:MaxGCPauseMillis=200 \
    -XX:+UnlockExperimentalVMOptions \
    -XX:+DisableExplicitGC \
    -XX:+AlwaysPreTouch \
    -XX:G1NewSizePercent=30 \
    -XX:G1MaxNewSizePercent=40 \
    -XX:G1HeapRegionSize=8M \
    -XX:G1ReservePercent=20 \
    -XX:G1HeapWastePercent=5 \
    -XX:G1MixedGCCountTarget=4 \
    -XX:InitiatingHeapOccupancyPercent=15 \
    -XX:G1MixedGCLiveThresholdPercent=90 \
    -XX:G1RSetUpdatingPauseTimePercent=5 \
    -XX:SurvivorRatio=32 \
    -XX:+PerfDisableSharedMem \
    -XX:MaxTenuringThreshold=1 \
    -jar server.jar --nogui
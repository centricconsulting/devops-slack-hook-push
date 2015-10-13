# Usage: deploy.sh [dev|qa|demo|stage|prod] [app_name] [port]

# Pull the repo down.
echo "Pulling Spicoli repository..."
docker pull centric/slack-hook-push:$1
# Stop the currently executing container.
echo "Stopping any running containers..."
docker stop spicoli_$1
# Remove all of the orphaned containers.
echo "Remove all orphaned and exited containers..."
docker rm $(docker ps -q -f status=exited)
# Start the application.
echo "Starting the application..."
docker run -d -p $3:1966 --name $2_$1 centric/slack-hook-push:$1

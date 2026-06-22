#!/bin/sh
# SeaweedFS S3 bucket setup: enable anonymous read for public asset access
set -e

echo "Waiting for SeaweedFS master on port 9333..."
for i in $(seq 1 30); do
  if curl -s http://seaweedfs:9333/cluster/healthz > /dev/null 2>&1; then
    echo "SeaweedFS is ready."
    break
  fi
  echo "  attempt $i/30..."
  sleep 2
done

echo "Configuring S3 buckets for anonymous read..."

# Create buckets if they don't exist and enable anonymous read
for bucket in opendsp-creatives opendsp-proofs opendsp-assets; do
  echo "  Setting up bucket: $bucket"
  echo "s3.bucket.create -name $bucket" | weed shell -master=seaweedfs:9333 2>/dev/null || true
  echo "s3.configure -user=dummy -actions=Read,Write,List,Tagging,Admin -buckets=$bucket -apply" | weed shell -master=seaweedfs:9333 2>/dev/null || true
done

# Enable anonymous read on all three buckets
echo "s3.configure -actions=Read -buckets=opendsp-creatives,opendsp-proofs,opendsp-assets -apply" | weed shell -master=seaweedfs:9333 2>/dev/null || true

echo "SeaweedFS S3 bucket setup complete."

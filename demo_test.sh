#!/bin/bash

# Steer Demo Automation Test Script
# =================================
# This script automates the functional testing of the Steer demo.
# It starts the backend server, creates resources via API, waits for processing,
# and verifies the results.
#
# Workflow:
# 1. Start the backend server in the background
# 2. Wait for the server to be ready
# 3. Create a HelmRelease
# 4. Create a HelmTestJob with delay and hooks
# 5. Poll the job status until completion
# 6. Verify the test results and hook execution
# 7. Clean up and stop the server

# Configuration
API_URL="http://localhost:8080/api/v1"
SERVER_PID=""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[TEST]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    cleanup
    exit 1
}

cleanup() {
    if [ -n "$SERVER_PID" ]; then
        log "Stopping backend server (PID: $SERVER_PID)..."
        kill $SERVER_PID 2>/dev/null
    fi
}

# Trap interrupts
trap cleanup EXIT INT TERM

# 1. Start Backend Server
log "Starting backend server..."
cd backend
go run main.go > server.log 2>&1 &
SERVER_PID=$!
cd ..

log "Server started with PID $SERVER_PID. Waiting for readiness..."

# 2. Wait for Server Readiness
for i in {1..30}; do
    if curl -s "$API_URL/helmreleases" >/dev/null; then
        log "Server is ready!"
        break
    fi
    sleep 1
    echo -n "."
done

if ! curl -s "$API_URL/helmreleases" >/dev/null; then
    error "Server failed to start within 30 seconds. Check backend/server.log"
fi

# 3. Create HelmRelease
log "Creating HelmRelease 'nginx-demo'..."
curl -s -X POST "$API_URL/helmreleases" \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
        "name": "nginx-demo",
        "namespace": "default"
    },
    "spec": {
        "chart": {
            "name": "nginx",
            "version": "1.0.0"
        },
        "deployment": {
            "namespace": "test-ns"
        }
    },
    "status": {
        "phase": "Pending"
    }
}' | grep "nginx-demo" >/dev/null || error "Failed to create HelmRelease"

log "HelmRelease created. Waiting for deployment..."

# Poll HelmRelease Status
MAX_RELEASE_RETRIES=10
RELEASE_READY=false

for i in $(seq 1 $MAX_RELEASE_RETRIES); do
    STATUS=$(curl -s "$API_URL/helmreleases/default/nginx-demo" | jq -r '.status.phase')
    echo "Release Status: $STATUS"
    
    if [ "$STATUS" == "Installed" ]; then
        RELEASE_READY=true
        log "HelmRelease deployed successfully (Status: $STATUS)"
        break
    elif [ "$STATUS" == "Failed" ]; then
        error "HelmRelease deployment failed"
    fi
    
    sleep 2
done

if [ "$RELEASE_READY" = false ]; then
    error "Timeout waiting for HelmRelease deployment"
fi

# 4. Create HelmTestJob
log "Creating HelmTestJob 'test-job-01' with delay and hooks..."
curl -s -X POST "$API_URL/helmtestjobs" \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
        "name": "test-job-01",
        "namespace": "default"
    },
    "spec": {
        "helmReleaseRef": {
            "name": "nginx-demo",
            "namespace": "default"
        },
        "schedule": {
            "type": "once",
            "delay": "2s"
        },
        "test": {
            "timeout": "5m"
        },
        "hooks": {
            "preTest": [
                {
                    "name": "check-env",
                    "type": "script",
                    "env": [
                        {
                            "name": "RELEASE_NAME",
                            "valueFrom": {
                                "helmReleaseRef": {
                                    "fieldPath": "metadata.name"
                                }
                            }
                        }
                    ],
                    "script": "echo Checking release $RELEASE_NAME"
                }
            ]
        }
    },
    "status": {
        "phase": "Pending"
    }
}' | grep "test-job-01" >/dev/null || error "Failed to create HelmTestJob"

log "HelmTestJob created. Waiting for execution (including delay)..."

# 5. Poll Job Status
MAX_RETRIES=20
for i in $(seq 1 $MAX_RETRIES); do
    RESPONSE=$(curl -s "$API_URL/helmtestjobs/default/test-job-01")
    PHASE=$(echo $RESPONSE | jq -r '.status.phase')
    
    echo "Current Phase: $PHASE"
    
    if [ "$PHASE" == "Succeeded" ]; then
        log "Job completed successfully!"
        
        # 6. Verify Results
        # Check Test Results
        TEST_RESULT=$(echo $RESPONSE | jq -r '.status.testResults[0].phase')
        if [ "$TEST_RESULT" != "Succeeded" ]; then
            error "Test result is '$TEST_RESULT', expected 'Succeeded'"
        fi
        log "Test execution passed"
        
        # Check Hook Results
        HOOK_RESULT=$(echo $RESPONSE | jq -r '.status.hookResults[0].phase')
        if [ "$HOOK_RESULT" != "Succeeded" ]; then
            error "Hook result is '$HOOK_RESULT', expected 'Succeeded'"
        fi
        log "Hook execution passed"
        
        break
    elif [ "$PHASE" == "Failed" ]; then
        error "Job failed!"
    fi
    
    if [ $i -eq $MAX_RETRIES ]; then
        error "Timeout waiting for job completion"
    fi
    
    sleep 2
done

log "All tests passed successfully!"
echo -e "${YELLOW}Demo test completed.${NC}"

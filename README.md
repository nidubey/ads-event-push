# ads-event-push

## Script originally written by Brian and copied from here
https://segment.atlassian.net/wiki/spaces/PERSONAS/pages/1687683645/Testing+Batch+Integrations

## To push the event run:
go run main.go --writeKey={write_key_from_your_segment_source} --numUsers=10 --maxConcurrent=1 --env=stage --eventType=track --debug=true
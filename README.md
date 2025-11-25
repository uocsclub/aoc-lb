# Quick overview

This is an app to host a private advent of code leaderboard with custom language modifiers

This app is built in go using gofiber, tailwind, htmx and templ

# Env variables

```
SESSION_ID=<AOC session cookie (required to fetch their API)>
SERVER_PORT=<Port to run the server on (set it to 7071 (yes, not 7070))>
YEAR=<Year to fetch the leaderboard for>
LEADERBOARD_ID=<ID of the private leaderboard>
GITHUB_OAUTH_ID=<ID of your github oauth integration>
GITHUB_OAUTH_REDIRECT_URI=<Base redirect urI for your github oauth integration>
GITHUB_OAUTH_SECRET=<Secret for your github oauth integration>
```

# Dev
For development purposes, simply run `make` and it will run on port 7070 (yes, not 7071)

# For prod deployment

There is a Dockerfile which contains the prod build, just deploy that using whatever way you want

# Scripts

**./create_migration.sh**

Takes in 1 arg that is the name of the migration, and it creates an up and down migration in ./migrations


#! /bin/bash

## No need to do this as we are already checked-out via actions
# git clone git@github.com:external-secrets-inc/external-secrets-enterprise
# cd external-secrets-enterprise

git fetch upstream

git fetch origin

git checkout origin/main

git checkout -b update-upstream-$(date +%s)

git merge upstream/main --no-ff --no-commit || true

#verify if any files were added. If they were, commit them. If not, skip safely
if [[ -z "$(git status --porcelain)" ]]; then
  echo "nothing changed. skipping."
  exit 0;
fi

git add .

## Remove .github folder from the commit
git reset HEAD .github
git checkout .github

git commit -s -m "chore: update upstream"

git push origin update-upstream-$(date +%s)

gh pr create --title "chore: update upstream" --body "update upstream" --base main --head update-upstream-$(date +%s)
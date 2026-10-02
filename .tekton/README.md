# Konflux build failure alerts

The four PipelineRuns in this directory build the applier image and Helm chart:

| PipelineRun | Trigger |
| --- | --- |
| `hyperfleet-applier-on-push` | Push to `main` |
| `hyperfleet-applier-on-tag` | Version tag |
| `hyperfleet-applier-chart-on-push` | Push to `main` |
| `hyperfleet-applier-chart-on-tag` | Version tag |

Each inline pipeline has a `slack-webhook-notification` `finally` task guarded by `$(tasks.status) in ["Failed"]`. A completed failed build names the pipeline and links its repository, commit, and failed run in `#hyperfleet-e2e-status`. A successful build does not run this task. PaC admission failures and runs that never start cannot reach a `finally` task. Release notifications use a separate mechanism.

The task reads key `hyperfleet-slack-webhook-url` from Secret `hyperfleet-slack-webhook-notification-secret` in `hyperfleet-tenant`. HyperFleet manages the Secret, alert behavior, and channel. The webhook URL must never be stored in this repository or included in logs or tickets. The image builds run as `build-pipeline-hyperfleet-applier`; chart builds run as `build-pipeline-hyperfleet-applier-chart`. Both service accounts need to be able to mount the tenant Secret.

## Check an alert

Open the failed PipelineRun in [Konflux](https://konflux-ui.apps.kflux-prd-rh02.0fk9.p1.openshiftapps.com/) and inspect its `slack-webhook-notification` final task. If the task did not start, check the run status and its `when` condition. If it failed, inspect its TaskRun logs for Secret mount, webhook, network, or Slack errors without printing the webhook value. Compare the message's pipeline, commit, and run link with that PipelineRun. A missing PaC run requires [trigger debugging](https://github.com/openshift-hyperfleet/architecture/blob/main/hyperfleet/docs/release/operations/debugging.md#i-pushed-a-tag-and-no-pipelinerun-started).

With authorization for a disposable failing run and Slack post, verify that one failed build posts promptly with all fields and a working link. Observe an updated successful run and confirm this build failure task is skipped and no build failure alert appears. A successful tag can also trigger a release with its own notification; distinguish that message from this one. Static YAML checks do not establish delivery or timing.

## Rotate the webhook

1. Coordinate with the HyperFleet team to create a replacement incoming webhook for `#hyperfleet-e2e-status`.
2. Update `hyperfleet-tenant/hyperfleet-slack-webhook-notification-secret`, key `hyperfleet-slack-webhook-url`, through the team's Secret management process and update its configuration source if one is used. If the same webhook also serves release notifications, coordinate the release source and Secret update with RelEng.
3. With authorization, run a controlled failed build and confirm its alert and link. If the webhook is shared, also verify a release notification.
4. Revoke the old webhook only after every consumer has been verified on the replacement.

See the [HyperFleet notification runbook](https://github.com/openshift-hyperfleet/architecture/blob/main/hyperfleet/docs/release/operations/notifications.md) for the separate release notification path and escalation context.

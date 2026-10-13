# internal/reconciler

`Runnable` and `Reconcilable` are the two lifecycles the applier starts.

```mermaid
flowchart TB
    apply["ApplyReconciler"]
    delete["DeleteReconciler"]
    reconcilable["reconciler.Reconcilable"]
    serial["serial.Runnable"]
    read["readdesire.Controller"]
    refresher["discovery.Refresher"]
    runnable["reconciler.Runnable"]
    start["startRunnables"]

    apply -->|"implements"| reconcilable
    delete -->|"implements"| reconcilable
    reconcilable -->|"serial.New stores them"| serial

    serial -->|"implements"| runnable
    read -->|"implements"| runnable
    refresher -->|"implements"| runnable
    runnable -->|"Start"| start
```

`serial.New` returns `*serial.Runnable`. `readdesire.New` returns `*Controller`. `discovery.New` returns `*Refresher`. All three implement `Runnable`, and `startRunnables` calls `Start` on each. `serial.Runnable.Start` then calls `RunOnce` on apply, then delete.

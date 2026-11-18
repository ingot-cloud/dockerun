#!/usr/bin/env python3
"""Re-run cases that failed due to test-helper compose/publish payload."""

from run_release_workspace_tests import (
    encrypt_password, http, record, results, save, run, wait_deployment, CLI, BASE, ADMIN_PASSWORD,
)
import run_release_workspace_tests as t
import json
import time
import subprocess
import sys

login = http("POST", "/api/auth/login", {"username": "admin", "password": encrypt_password(ADMIN_PASSWORD)}, auth=False)
if login.get("code") != 200:
    print(login)
    sys.exit(1)
t.token = login["body"]["token"]
tok = http("POST", "/api/auth/tokens", {"name": "remaining-2222", "scopes": ["pull", "push", "deploy"], "days": 7})
cli_token = tok["body"]["token"] if tok.get("code") == 201 else ""

compose = (
    "services:\n"
    "  demo:\n"
    "    image: 127.0.0.1:18080/dockerun-test/demo:v1\n"
    "    restart: unless-stopped\n"
    "    command: [sh, -c, 'while true; do sleep 30; done']\n"
    "    healthcheck:\n"
    "      test: [CMD, 'true']\n"
    "      interval: 1s\n"
    "      timeout: 1s\n"
    "      retries: 5\n"
)

saved = http("PUT", "/api/applications/1/deployment-config", {
    "environment_id": 1,
    "compose": compose,
    "configs": {},
    "execution": {},
    "scripts": {},
    "source": "registry",
    "registry_id": 1,
    "platform": "linux/amd64",
})
save("s4_save_config_retry.json", saved)
record("S4.1 save deployment config", "pass" if saved.get("code") == 200 else "fail", **saved)

release = http("POST", "/api/applications/1/releases?environment_id=1", {
    "version": "v1-registry-test",
    "source": "registry",
    "registry_id": 1,
    "compose": compose,
    "configs": "{}",
    "platform": "linux/amd64",
    "save_definition": True,
})
save("s4_create_release_retry.json", release)
record("S4.2 create registry release", "pass" if release.get("code") == 201 else "fail", **release)
release_id = (release.get("body") or {}).get("id")

if release_id:
    pre = http("POST", "/api/deployments/preflight", {"environment_id": 1, "release_id": release_id})
    save("s4_preflight_retry.json", pre)
    record("S4.3 preflight", "pass" if pre.get("code") == 200 else "fail", **pre)
    dep = http("POST", "/api/deployments", {
        "environment_id": 1,
        "release_id": release_id,
        "idempotency_key": f"manual-retry-{int(time.time()*1000)}",
    })
    save("s4_enqueue_retry.json", dep)
    record("S4.4 enqueue deploy", "pass" if dep.get("code") == 202 else "fail", **dep)
    if dep.get("code") == 202:
        waited = wait_deployment(dep["body"]["id"])
        save("s4_deploy_wait_retry.json", waited)
        status = ((waited.get("body") or {}).get("deployment") or {}).get("status")
        record("S4.4 deploy result", "pass" if status == "success" else "fail", deploy_status=status, result=((waited.get("body") or {}).get("deployment") or {}).get("result"))
        detail = http("GET", f"/api/deployments/{dep['body']['id']}")
        save("s5_deployment_detail.json", detail)
        record("S5.2 deployment detail", "pass" if detail.get("code") == 200 else "fail", code=detail.get("code"))
        if cli_token:
            stid = run([CLI, "status", str(dep["body"]["id"])], env={"DOCKERUN_URL": BASE, "DOCKERUN_TOKEN": cli_token})
            save("s5_cli_status_id.txt", stid)
            record("S5.3 CLI status id", "pass" if stid["code"] == 0 else "fail", code=stid["code"])

auto = http("PATCH", "/api/applications/1/environments/1/automation", {
    "auto_deploy": True,
    "registry_id": 1,
    "repository": "dockerun-test/demo",
    "tag_pattern": "v*",
    "allowed_publishers": "[]",
})
save("s4_automation_retry.json", auto)
record("S4.5 enable auto-deploy", "pass" if auto.get("code") == 200 else "fail", **auto)

p = subprocess.run(["docker", "tag", "nginx:1.28-alpine", "127.0.0.1:18080/dockerun-test/demo:v3"], capture_output=True, text=True)
p2 = subprocess.run(["docker", "push", "127.0.0.1:18080/dockerun-test/demo:v3"], capture_output=True, text=True)
save("s4_push_v3.json", {"tag": p.returncode, "push": p2.returncode, "stdout": p2.stdout, "stderr": p2.stderr})
record("S4.5 push builtin demo:v3", "pass" if p2.returncode == 0 else "fail", code=p2.returncode)

auto_dep = None
deadline = time.time() + 90
while time.time() < deadline:
    listed = http("GET", "/api/deployments")
    if listed.get("code") == 200:
        for row in listed["body"] or []:
            if row.get("automatic") and row.get("environment_id") == 1:
                auto_dep = row
                break
    if auto_dep:
        break
    time.sleep(2)
save("s4_auto_deployments_retry.json", auto_dep or {"note": "none"})
if auto_dep:
    waited = wait_deployment(auto_dep["id"])
    save("s4_auto_wait_retry.json", waited)
    status = ((waited.get("body") or {}).get("deployment") or {}).get("status")
    record("S4.5 auto-deploy from builtin push", "pass" if status == "success" else "fail", deploy_status=status, deployment_id=auto_dep["id"], result=((waited.get("body") or {}).get("deployment") or {}).get("result"))
else:
    record("S4.5 auto-deploy from builtin push", "fail", note="90s 内未见 automatic 部署任务")

tmpl = http("POST", "/api/applications/1/template", {
    "name": "custom-demo-2222",
    "version": "1.0.0",
    "description": "由空白应用另存",
})
save("s3_save_custom_template_retry.json", tmpl)
record("S3.3 save custom template", "pass" if tmpl.get("code") == 201 else "fail", **tmpl)
listed_t = http("GET", "/api/managed-templates")
custom_ids = [x.get("id") for x in (listed_t.get("body") or []) if str(x.get("id", "")).startswith("custom-")]
save("s3_templates_after_retry.json", listed_t)
record("S3.4 custom template listed", "pass" if custom_ids else "fail", custom_ids=custom_ids)

# Bundle publish needs the offline release id (source=bundle). Release #1 from first run.
pub = http("POST", "/api/bundles/publish", {"release_id": 1})
save("s6_publish_retry.json", pub)
record("S6.5 publish bundle to builtin", "pass" if pub.get("code") == 201 else "fail", **pub)

save("results_remaining.json", results)
failed = [r for r in results if r["status"] == "fail"]
print(f"REMAINING pass={sum(1 for r in results if r['status']=='pass')} fail={len(failed)} skip={sum(1 for r in results if r['status']=='skip')}", flush=True)
sys.exit(1 if failed else 0)

#!/usr/bin/env python3
from run_release_workspace_tests import (
    encrypt_password, http, record, results, save, run, wait_deployment, CLI, BASE, ADMIN_PASSWORD,
)
import run_release_workspace_tests as t
import subprocess, sys, time

login = http("POST", "/api/auth/login", {"username": "admin", "password": encrypt_password(ADMIN_PASSWORD)}, auth=False)
t.token = login["body"]["token"]
tok = http("POST", "/api/auth/tokens", {"name": "leftover-2222", "scopes": ["pull", "push", "deploy"], "days": 7})
cli_token = tok["body"]["token"]

detail = http("GET", "/api/deployments/2")
save("s5_deployment_detail.json", detail)
record("S5.2 deployment detail", "pass" if detail.get("code") == 200 and (detail["body"]["deployment"]["status"] == "success") else "fail", code=detail.get("code"), deploy_status=(detail.get("body") or {}).get("deployment", {}).get("status"))
stid = run([CLI, "status", "2"], env={"DOCKERUN_URL": BASE, "DOCKERUN_TOKEN": cli_token})
save("s5_cli_status_id.txt", stid)
record("S5.3 CLI status id", "pass" if stid["code"] == 0 else "fail", code=stid["code"])

auto = http("PATCH", "/api/applications/1/environments/1/automation", {
    "auto_deploy": True, "registry_id": 1, "repository": "dockerun-test/demo", "tag_pattern": "v*", "allowed_publishers": "[]",
})
save("s4_automation_retry.json", auto)
record("S4.5 enable auto-deploy", "pass" if auto.get("code") == 200 else "fail", **auto)

before = http("GET", "/api/deployments")
before_ids = {row["id"] for row in (before.get("body") or [])}
p = subprocess.run(["docker", "tag", "nginx:1.28-alpine", "127.0.0.1:18080/dockerun-test/demo:v3"], capture_output=True, text=True)
p2 = subprocess.run(["docker", "push", "127.0.0.1:18080/dockerun-test/demo:v3"], capture_output=True, text=True)
save("s4_push_v3.json", {"tag": p.returncode, "push": p2.returncode, "stdout": p2.stdout[-2000:], "stderr": p2.stderr})
record("S4.5 push builtin demo:v3", "pass" if p2.returncode == 0 else "fail", code=p2.returncode)

auto_dep = None
deadline = time.time() + 90
while time.time() < deadline:
    listed = http("GET", "/api/deployments")
    if listed.get("code") == 200:
        for row in listed["body"] or []:
            if row.get("id") not in before_ids and row.get("automatic"):
                auto_dep = row
                break
    if auto_dep:
        break
    time.sleep(2)
save("s4_auto_deployments_retry.json", auto_dep or {"note": "none", "listed": before.get("body")})
if auto_dep:
    waited = wait_deployment(auto_dep["id"])
    save("s4_auto_wait_retry.json", waited)
    deploy_status = ((waited.get("body") or {}).get("deployment") or {}).get("status")
    record("S4.5 auto-deploy from builtin push", "pass" if deploy_status == "success" else "fail", deploy_status=deploy_status, deployment_id=auto_dep["id"], result=((waited.get("body") or {}).get("deployment") or {}).get("result"))
else:
    overview = http("GET", "/api/overview")
    catalog = http("GET", "/api/builtin/catalog")
    save("s4_auto_missing_retry.json", {"overview": overview, "catalog": catalog, "deployments": listed if "listed" in dir() else before})
    record("S4.5 auto-deploy from builtin push", "fail", note="90s 内未见新 automatic 部署任务")

tmpl = http("POST", "/api/applications/1/template", {"name": "custom-demo-2222", "version": "1.0.0", "description": "由空白应用另存"})
save("s3_save_custom_template_retry.json", tmpl)
record("S3.3 save custom template", "pass" if tmpl.get("code") == 201 else "fail", **tmpl)
listed_t = http("GET", "/api/managed-templates")
custom_ids = [x.get("id") for x in (listed_t.get("body") or []) if str(x.get("id", "")).startswith("custom-")]
save("s3_templates_after_retry.json", listed_t)
record("S3.4 custom template listed", "pass" if custom_ids else "fail", custom_ids=custom_ids)

pub = http("POST", "/api/bundles/publish", {"release_id": 1})
save("s6_publish_retry.json", pub)
record("S6.5 publish bundle to builtin", "pass" if pub.get("code") == 201 else "fail", **pub)

save("results_leftover.json", results)
failed = [r for r in results if r["status"] == "fail"]
print(f"LEFTOVER pass={sum(1 for r in results if r['status']=='pass')} fail={len(failed)}", flush=True)
sys.exit(1 if failed else 0)

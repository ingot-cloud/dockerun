#!/usr/bin/env python3
"""Three Docker-level release scenarios. Test helper only; no product code changes."""

from __future__ import annotations

import hashlib
import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

BASE = os.environ.get("DOCKERUN_URL", "http://127.0.0.1:18080").rstrip("/")
EVIDENCE = Path(os.environ.get("EVIDENCE_DIR", "/workspace/artifacts/evidence/scenario"))
CLI = os.environ.get("DOCKERUN_CLI", "/workspace/bin/dockerun")
ADMIN_PASSWORD = os.environ.get("DOCKERUN_ADMIN_PASSWORD", "TestAdmin2222!")
WORK = Path(os.environ.get("SCENARIO_WORK", "/tmp/dockerun-scenario-work-2222"))
REG = "127.0.0.1:18080/dockerun-scenario/demo"
C1 = "dockerun-sc1-app-2222"
C3 = "dockerun-sc3-app-2222"
TIMEOUT = 180

EVIDENCE.mkdir(parents=True, exist_ok=True)
WORK.mkdir(parents=True, exist_ok=True)

results: list[dict] = []
token = ""
state: dict = {}


def encrypt_password(password: str) -> str:
    key = ("dockerun-aes-key-2025-secret-key".ljust(32, "0")[:32]).encode()
    iv = ("dockerun-aes-iv16".ljust(16, "0")[:16]).encode()
    return subprocess.check_output(
        ["openssl", "enc", "-aes-256-cbc", "-K", key.hex(), "-iv", iv.hex(), "-base64", "-A"],
        input=password.encode(),
    ).decode().strip()


def record(case: str, status: str, **extra):
    row = {"case": case, "status": status, "ts": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), **extra}
    results.append(row)
    print(json.dumps(row, ensure_ascii=False), flush=True)
    return row


def save(name: str, payload):
    path = EVIDENCE / name
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2, default=str) + "\n", encoding="utf-8")
    return str(path)


def sh(cmd, input_text=None, timeout=180, check=False):
    p = subprocess.run(cmd, input=input_text, capture_output=True, text=True, timeout=timeout)
    out = {"cmd": cmd, "code": p.returncode, "stdout": p.stdout[-12000:], "stderr": p.stderr[-4000:]}
    if check and p.returncode != 0:
        raise RuntimeError(json.dumps(out, ensure_ascii=False))
    return out


def http(method: str, path: str, body=None, auth=True, headers=None, timeout=TIMEOUT):
    url = path if path.startswith("http") else BASE + path
    data = None
    req_headers = {"Accept": "application/json"}
    if body is not None and not isinstance(body, (bytes, bytearray)):
        data = json.dumps(body).encode()
        req_headers["Content-Type"] = "application/json"
    elif isinstance(body, (bytes, bytearray)):
        data = body
    if headers:
        req_headers.update(headers)
    if auth and token:
        req_headers["Authorization"] = "Bearer " + token
    req = urllib.request.Request(url, data=data, method=method, headers=req_headers)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read()
            parsed = json.loads(raw.decode() or "null") if raw else None
            return {"code": resp.status, "body": parsed}
    except urllib.error.HTTPError as e:
        raw = e.read()
        try:
            parsed = json.loads(raw.decode() or "null")
        except Exception:
            parsed = raw.decode("utf-8", "replace")
        return {"code": e.code, "body": parsed}
    except Exception as e:
        return {"code": 0, "error": str(e)}


def docker_inspect(ref):
    p = sh(["docker", "inspect", ref])
    if p["code"] != 0:
        return None
    try:
        return json.loads(p["stdout"])[0]
    except Exception:
        return None


def image_id(ref):
    info = docker_inspect(ref)
    return info.get("Id") if info else None


def container_snapshot(name):
    info = docker_inspect(name)
    if not info:
        return {"exists": False, "name": name}
    state = info.get("State") or {}
    return {
        "exists": True,
        "name": name,
        "id": info.get("Id"),
        "image": info.get("Config", {}).get("Image"),
        "image_id": info.get("Image"),
        "running": bool(state.get("Running")),
        "status": state.get("Status"),
        "started_at": state.get("StartedAt"),
    }


def image_snapshot(ref):
    info = docker_inspect(ref)
    if not info:
        return {"exists": False, "ref": ref}
    repo = info.get("RepoDigests") or []
    return {"exists": True, "ref": ref, "id": info.get("Id"), "repo_digests": repo, "labels": (info.get("Config") or {}).get("Labels")}


def wait_health(seconds=60):
    last = None
    deadline = time.time() + seconds
    while time.time() < deadline:
        last = http("GET", "/health", auth=False)
        if last.get("code") == 200:
            return last
        time.sleep(0.5)
    return last or {"code": 0, "error": "timeout"}


def wait_deployment(dep_id, seconds=180):
    last = None
    deadline = time.time() + seconds
    while time.time() < deadline:
        last = http("GET", f"/api/deployments/{dep_id}")
        if last.get("code") == 200:
            status = ((last.get("body") or {}).get("deployment") or {}).get("status")
            if status in {"success", "failed", "cancelled", "needs_review"}:
                return last
        time.sleep(1.5)
    return last or {"code": 0, "error": "timeout"}


def wait_auto(env_id, before_ids, seconds=90):
    deadline = time.time() + seconds
    listed = None
    while time.time() < deadline:
        listed = http("GET", "/api/deployments")
        if listed.get("code") == 200:
            for row in listed["body"] or []:
                if row.get("id") not in before_ids and row.get("automatic") and row.get("environment_id") == env_id:
                    return wait_deployment(row["id"])
        time.sleep(2)
    return {"code": 0, "error": "no automatic deployment", "listed": listed}


def wait_bundle(bundle_id, seconds=120):
    deadline = time.time() + seconds
    while time.time() < deadline:
        listed = http("GET", "/api/bundles")
        if listed.get("code") == 200:
            for row in listed["body"] or []:
                if row.get("id") == bundle_id:
                    if row.get("status") in {"ready", "failed"}:
                        return row
        time.sleep(1)
    return {"status": "timeout", "id": bundle_id}


def upload_file(path: Path, kind: str):
    data = path.read_bytes()
    digest = hashlib.sha256(data).hexdigest()
    st = http("GET", f"/api/bundles/uploads/{digest}")
    if st.get("code") != 200:
        return {"error": "offset", **st}
    offset = (st.get("body") or {}).get("offset") or 0
    chunk = 8 * 1024 * 1024
    while offset < len(data):
        end = min(offset + chunk, len(data))
        put = http(
            "PUT",
            f"/api/bundles/uploads/{digest}",
            body=data[offset:end],
            headers={"Content-Type": "application/octet-stream", "Content-Range": f"bytes {offset}-{end - 1}/{len(data)}"},
        )
        if put.get("code") != 200:
            return {"error": "put", **put}
        offset = (put.get("body") or {}).get("offset")
        if not offset:
            return {"error": "no offset", **put}
    done = http("POST", f"/api/bundles/uploads/{digest}/complete", {"kind": kind, "name": path.name})
    return done


def log_text(detail):
    logs = ((detail.get("body") or {}).get("logs") or [])
    return [f"[{x.get('stage')}] {x.get('message')}" for x in logs]


def assert_log_sequence(messages, needles):
    idx = 0
    found = []
    for needle in needles:
        hit = None
        for i in range(idx, len(messages)):
            if needle in messages[i]:
                hit = i
                break
        if hit is None:
            return False, found
        found.append({"needle": needle, "index": hit, "line": messages[hit]})
        idx = hit + 1
    return True, found


def setup_auth():
    global token
    setup = http("GET", "/api/auth/check-setup", auth=False)
    if setup.get("code") == 200 and (setup.get("body") or {}).get("require_setup"):
        created = http("POST", "/api/auth/setup", {"new_password": encrypt_password(ADMIN_PASSWORD)}, auth=False)
        record("S0 setup", "pass" if created.get("code") == 200 else "fail", code=created.get("code"))
    login = http("POST", "/api/auth/login", {"username": "admin", "password": encrypt_password(ADMIN_PASSWORD)}, auth=False)
    if login.get("code") != 200:
        record("S0 login", "fail", **login)
        return False
    token = login["body"]["token"]
    tok = http("POST", "/api/auth/tokens", {"name": "scenario-2222", "scopes": ["pull", "push", "deploy"], "days": 7})
    if tok.get("code") != 201:
        record("S0 token", "fail", **tok)
        return False
    state["cli_token"] = tok["body"]["token"]
    record("S0 login+token", "pass", code=201)
    return True


def docker_login():
    p = subprocess.run(
        ["docker", "login", "-u", "admin", "--password-stdin", "127.0.0.1:18080"],
        input=state["cli_token"],
        capture_output=True,
        text=True,
        timeout=60,
    )
    save("s0_docker_login.json", {"code": p.returncode, "stdout": p.stdout, "stderr": p.stderr})
    record("S0 docker login", "pass" if p.returncode == 0 else "fail", code=p.returncode)
    return p.returncode == 0


def build_images():
    for key, text in (("a", "IMAGE_A"), ("b", "IMAGE_B")):
        df = WORK / f"Dockerfile.{key}"
        df.write_text(
            "FROM alpine:3.20\n"
            f"LABEL dockerun.scenario={key} dockerun.fixture=scenario-2222\n"
            f"RUN echo {text} > /marker\n"
            f'CMD ["sh", "-c", "echo {text}; sleep infinity"]\n',
            encoding="utf-8",
        )
        built = sh(["docker", "build", "-f", str(df), "-t", f"dockerun-scenario-{key}:local", str(WORK)], timeout=180)
        save(f"build_{key}.json", built)
        if built["code"] != 0:
            record(f"S0 build image {key}", "fail", **built)
            return False
        record(f"S0 build image {key}", "pass", image_id=image_id(f"dockerun-scenario-{key}:local"))
    state["id_a"] = image_id("dockerun-scenario-a:local")
    state["id_b"] = image_id("dockerun-scenario-b:local")
    if not state["id_a"] or not state["id_b"] or state["id_a"] == state["id_b"]:
        record("S0 distinct image IDs", "fail", id_a=state.get("id_a"), id_b=state.get("id_b"))
        return False
    record("S0 distinct image IDs", "pass", id_a=state["id_a"], id_b=state["id_b"])
    return True


def push_tag(local, tag):
    remote = f"{REG}:{tag}"
    sh(["docker", "tag", local, remote], check=True)
    pushed = sh(["docker", "push", remote], timeout=180)
    save(f"push_{tag}.json", pushed)
    return pushed["code"] == 0, remote


def create_command_app(name, container, extra_params=None):
    app = http("POST", "/api/applications", {"name": name, "description": "scenario command app"})
    if app.get("code") != 201:
        return None
    app_id = app["body"]["id"]
    env = http("PUT", f"/api/applications/{app_id}/environments", {
        "name": "test",
        "target_id": 1,
        "registry_id": 1,
        "parameters": {"APP_NAME": container, **(extra_params or {})},
        "stateless": True,
        "script_mode": "inherit",
        "target_ids": "[]",
        "allowed_publishers": "[]",
        "tag_pattern": "*",
    })
    if env.get("code") != 200:
        return None
    env_id = env["body"]["id"]
    cmd = (
        'docker run -d --name "$APP_NAME" --restart unless-stopped '
        '--label io.dockerun.application="$DOCKERUN_APPLICATION_ID" '
        '--label io.dockerun.environment="$DOCKERUN_ENVIRONMENT_ID" '
        '"$DOCKERUN_IMAGE"'
    )
    saved = http("PUT", f"/api/applications/{app_id}/deployment-config", {
        "environment_id": env_id,
        "compose": "",
        "configs": {},
        "execution": {"mode": "command", "command": cmd, "check_command": 'docker inspect -f "{{.State.Running}}" "$APP_NAME" | grep true'},
        "scripts": {},
        "source": "direct",
        "registry_id": 1,
        "platform": "linux/amd64",
        "container_names": ["$APP_NAME"],
        "replace_containers": True,
    })
    if saved.get("code") != 200:
        save(f"save_config_{name}.json", saved)
        return None
    auto = http("PATCH", f"/api/applications/{app_id}/environments/{env_id}/automation", {
        "auto_deploy": True,
        "registry_id": 1,
        "repository": "dockerun-scenario/demo",
        "tag_pattern": "*",
        "allowed_publishers": "[]",
        "container_names": ["$APP_NAME"],
        "replace_containers": True,
    })
    if auto.get("code") != 200:
        save(f"auto_{name}.json", auto)
        return None
    return {"app_id": app_id, "env_id": env_id, "name": name, "container": container}


def create_compose_app(name):
    app = http("POST", "/api/applications", {"name": name, "description": "scenario compose app"})
    if app.get("code") != 201:
        return None
    app_id = app["body"]["id"]
    env = http("PUT", f"/api/applications/{app_id}/environments", {
        "name": "test", "target_id": 1, "registry_id": 1, "parameters": {},
        "stateless": True, "script_mode": "inherit", "target_ids": "[]",
        "allowed_publishers": "[]", "tag_pattern": "*",
    })
    if env.get("code") != 200:
        return None
    return {"app_id": app_id, "env_id": env["body"]["id"], "name": name}


def compose_for(tag):
    return (
        "services:\n"
        "  demo:\n"
        f"    image: {REG}:{tag}\n"
        "    command: [sh, -c, 'sleep infinity']\n"
        "    labels:\n"
        "      dockerun.scenario: compose\n"
    )


def deploy_compose(app, tag, version):
    compose = compose_for(tag)
    saved = http("PUT", f"/api/applications/{app['app_id']}/deployment-config", {
        "environment_id": app["env_id"],
        "compose": compose,
        "configs": {},
        "execution": {},
        "scripts": {},
        "source": "registry",
        "registry_id": 1,
        "platform": "linux/amd64",
    })
    if saved.get("code") != 200:
        return None, saved
    rel = http("POST", f"/api/applications/{app['app_id']}/releases?environment_id={app['env_id']}", {
        "version": version,
        "source": "registry",
        "registry_id": 1,
        "compose": compose,
        "configs": "{}",
        "platform": "linux/amd64",
        "save_definition": True,
    })
    if rel.get("code") != 201:
        return None, rel
    pre = http("POST", "/api/deployments/preflight", {"environment_id": app["env_id"], "release_id": rel["body"]["id"]})
    dep = http("POST", "/api/deployments", {
        "environment_id": app["env_id"],
        "release_id": rel["body"]["id"],
        "idempotency_key": f"compose-{version}-{int(time.time()*1000)}",
    })
    if dep.get("code") != 202:
        return None, {"pre": pre, "dep": dep, "rel": rel}
    waited = wait_deployment(dep["body"]["id"])
    return waited, {"pre": pre, "rel": rel, "dep": dep, "wait": waited}


def compose_container_name(app):
    return f"dockerun-a{app['app_id']}-e{app['env_id']}-demo-1"


def scenario1(app):
    before = {row["id"] for row in (http("GET", "/api/deployments").get("body") or [])}
    ok, _ = push_tag("dockerun-scenario-a:local", "latest")
    if not ok:
        record("1.0 push A latest", "fail")
        return False
    first = wait_auto(app["env_id"], before)
    save("s1_first_deploy.json", first)
    status = ((first.get("body") or {}).get("deployment") or {}).get("status")
    if status != "success":
        record("1.0 baseline auto-deploy A:latest", "fail", deploy_status=status, result=((first.get("body") or {}).get("deployment") or {}).get("result"))
        return False
    snap0 = container_snapshot(C1)
    img0 = image_snapshot(f"{REG}:latest")
    save("s1_before_repush.json", {"container": snap0, "image": img0, "expected_id": state["id_a"]})
    if not snap0.get("running") or snap0.get("image_id") != state["id_a"]:
        record("1.0 baseline running A", "fail", snapshot=snap0, expected=state["id_a"])
        return False
    record("1.0 baseline auto-deploy A:latest", "pass", container_id=snap0["id"], image_id=snap0["image_id"])

    before = {row["id"] for row in (http("GET", "/api/deployments").get("body") or [])}
    ok, _ = push_tag("dockerun-scenario-b:local", "latest")
    if not ok:
        record("1.1 push B same tag latest", "fail")
        return False
    second = wait_auto(app["env_id"], before)
    save("s1_repush_deploy.json", second)
    status = ((second.get("body") or {}).get("deployment") or {}).get("status")
    logs = log_text(second)
    save("s1_repush_logs.json", logs)
    snap1 = container_snapshot(C1)
    img1 = image_snapshot(f"{REG}:latest")
    save("s1_after_repush.json", {"container": snap1, "image": img1})

    seq_ok, found = assert_log_sequence(logs, [
        "停止关联容器",
        "删除关联容器",
        "清理同 tag 的本地镜像引用",
    ])
    pulled = any(("拉取" in x) or ("Pull" in x) or ("摘要" in x) or ("已刷新本地 tag" in x) for x in logs)
    new_id = snap1.get("image_id")
    record("1.1 auto-deploy triggered", "pass" if status == "success" else "fail", deploy_status=status, deployment_id=((second.get("body") or {}).get("deployment") or {}).get("id"))
    record("1.2 stop then remove associated container (logs)", "pass" if seq_ok else "fail", found=found, logs=logs[:20])
    record("1.3 clean same-tag image ref then pull/retag", "pass" if seq_ok and pulled else "fail", pulled=pulled, logs=[x for x in logs if "images" in x or "清理" in x or "刷新" in x or "Pull" in x])
    record(
        "1.4 running Image ID changed to B",
        "pass" if snap1.get("running") and new_id == state["id_b"] and new_id != state["id_a"] and snap1.get("id") != snap0.get("id") else "fail",
        before=snap0,
        after=snap1,
        expected_b=state["id_b"],
    )
    return all(r["status"] == "pass" for r in results if r["case"].startswith("1."))


def scenario2(cmd_app, compose_app):
    # 2a auto: current is B@latest; push A as v2
    before = {row["id"] for row in (http("GET", "/api/deployments").get("body") or [])}
    snap0 = container_snapshot(C1)
    save("s2_before_v2.json", snap0)
    ok, _ = push_tag("dockerun-scenario-a:local", "v2")
    if not ok:
        record("2a.0 push A v2", "fail")
        return False
    auto = wait_auto(cmd_app["env_id"], before)
    save("s2_auto_v2.json", auto)
    logs = log_text(auto)
    save("s2_auto_v2_logs.json", logs)
    snap1 = container_snapshot(C1)
    img_v2 = image_snapshot(f"{REG}:v2")
    save("s2_after_v2.json", {"container": snap1, "image_v2": img_v2})
    status = ((auto.get("body") or {}).get("deployment") or {}).get("status")
    seq_ok, found = assert_log_sequence(logs, ["停止关联容器", "删除关联容器"])
    record("2a.1 auto-deploy different tag v2", "pass" if status == "success" else "fail", deploy_status=status)
    record("2a.2 stop current containers", "pass" if seq_ok and snap1.get("id") != snap0.get("id") else "fail", found=found, before_id=snap0.get("id"), after_id=snap1.get("id"))
    record(
        "2a.3 running Image ID matches tag v2 / image A",
        "pass" if snap1.get("running") and snap1.get("image_id") == state["id_a"] else "fail",
        running=snap1,
        expected=state["id_a"],
        tag_v2=img_v2,
    )

    # 2b manual compose: v1=A then latest=B
    ok, _ = push_tag("dockerun-scenario-a:local", "v1")
    first, ev = deploy_compose(compose_app, "v1", "manual-v1")
    save("s2_manual_v1.json", ev)
    cname = compose_container_name(compose_app)
    snap_c0 = container_snapshot(cname)
    save("s2_manual_v1_container.json", snap_c0)
    st = ((first.get("body") or {}).get("deployment") or {}).get("status") if first else None
    record("2b.1 manual compose deploy tag v1=A", "pass" if st == "success" and snap_c0.get("image_id") == state["id_a"] else "fail", deploy_status=st, snapshot=snap_c0)

    second, ev2 = deploy_compose(compose_app, "latest", "manual-latest-b")
    save("s2_manual_latest.json", ev2)
    snap_c1 = container_snapshot(cname)
    save("s2_manual_latest_container.json", snap_c1)
    st2 = ((second.get("body") or {}).get("deployment") or {}).get("status") if second else None
    logs2 = log_text(second) if second else []
    record("2b.2 manual compose deploy different tag latest=B", "pass" if st2 == "success" else "fail", deploy_status=st2)
    record(
        "2b.3 container recreated with Image ID of tag latest/B",
        "pass" if snap_c1.get("running") and snap_c1.get("image_id") == state["id_b"] and snap_c1.get("id") != snap_c0.get("id") else "fail",
        before=snap_c0,
        after=snap_c1,
        logs=logs2[-15:],
    )
    return True


def scenario3():
    app = create_command_app("scenario-local-2222", C3)
    save("s3_app.json", app)
    if not app:
        record("3.0 create local-import app", "fail")
        return False
    record("3.0 create local-import app", "pass", **app)

    # Bootstrap associated container with image A (not via dockerun).
    sh(["docker", "rm", "-f", C3])
    run = sh(["docker", "run", "-d", "--name", C3, "--restart", "unless-stopped", "dockerun-scenario-a:local"])
    save("s3_bootstrap.json", run)
    boot = container_snapshot(C3)
    if not boot.get("running") or boot.get("image_id") != state["id_a"]:
        record("3.0 bootstrap container A", "fail", snapshot=boot)
        return False
    record("3.0 bootstrap container A", "pass", snapshot=boot)

    tar = WORK / "image-b.tar"
    saved = sh(["docker", "save", "-o", str(tar), "dockerun-scenario-b:local"], timeout=120)
    if saved["code"] != 0 or not tar.exists():
        record("3.1 docker save B", "fail", **saved)
        return False
    record("3.1 docker save B", "pass", size=tar.stat().st_size)

    uploaded = upload_file(tar, "image")
    save("s3_upload.json", uploaded)
    if uploaded.get("code") not in {200, 201, 202}:
        record("3.2 upload kind=image", "fail", **uploaded)
        return False
    material = uploaded.get("body") or {}
    ready = wait_bundle(material.get("id"))
    save("s3_import_ready.json", ready)
    during = container_snapshot(C3)
    save("s3_container_during_import.json", during)
    record("3.2 import kind=image ready", "pass" if ready.get("status") == "ready" and ready.get("kind") == "image" else "fail", material=ready)
    record(
        "3.3 import does not stop existing container",
        "pass" if during.get("running") and during.get("id") == boot.get("id") and during.get("image_id") == state["id_a"] else "fail",
        before=boot,
        during=during,
    )

    rel = http("POST", f"/api/applications/{app['app_id']}/releases?environment_id={app['env_id']}", {
        "version": "local-b",
        "source": "local",
        "bundle_id": ready.get("id"),
        "execution": {
            "mode": "command",
            "command": (
                'docker run -d --name "$APP_NAME" --restart unless-stopped '
                '--label io.dockerun.application="$DOCKERUN_APPLICATION_ID" '
                '--label io.dockerun.environment="$DOCKERUN_ENVIRONMENT_ID" '
                '"$DOCKERUN_IMAGE"'
            ),
            "check_command": 'docker inspect -f "{{.State.Running}}" "$APP_NAME" | grep true',
        },
        "save_definition": True,
    })
    save("s3_release.json", rel)
    if rel.get("code") != 201:
        record("3.4 create source=local release from image archive", "fail", **rel)
        return False
    source = (rel.get("body") or {}).get("source")
    images = json.loads((rel.get("body") or {}).get("images") or "[]")
    record(
        "3.4 create source=local release from image archive",
        "pass" if source == "local" and images and images[0].get("id") == state["id_b"] else "fail",
        source=source,
        images=images,
        expected=state["id_b"],
    )
    pre = http("POST", "/api/deployments/preflight", {"environment_id": app["env_id"], "release_id": rel["body"]["id"]})
    save("s3_preflight.json", pre)
    still = container_snapshot(C3)
    record(
        "3.5 preflight does not stop container",
        "pass" if still.get("id") == boot.get("id") and still.get("running") else "fail",
        snapshot=still,
    )
    dep = http("POST", "/api/deployments", {
        "environment_id": app["env_id"],
        "release_id": rel["body"]["id"],
        "idempotency_key": f"local-{int(time.time()*1000)}",
    })
    save("s3_enqueue.json", dep)
    if dep.get("code") != 202:
        record("3.6 enqueue local deploy", "fail", **dep)
        return False
    waited = wait_deployment(dep["body"]["id"])
    save("s3_deploy.json", waited)
    logs = log_text(waited)
    save("s3_deploy_logs.json", logs)
    after = container_snapshot(C3)
    save("s3_after.json", after)
    status = ((waited.get("body") or {}).get("deployment") or {}).get("status")
    seq_ok, found = assert_log_sequence(logs, ["停止关联容器", "删除关联容器"])
    record("3.6 deploy from imported local Image ID", "pass" if status == "success" else "fail", deploy_status=status, result=((waited.get("body") or {}).get("deployment") or {}).get("result"))
    record("3.7 deploy stops associated container", "pass" if seq_ok and after.get("id") != boot.get("id") else "fail", found=found, before=boot["id"], after=after.get("id"))
    record(
        "3.8 running uses imported image B local ID",
        "pass" if after.get("running") and after.get("image_id") == state["id_b"] else "fail",
        after=after,
        expected=state["id_b"],
        path="POST /api/bundles/uploads/:sha/complete kind=image + POST /api/applications/:id/releases source=local bundle_id",
    )
    return True


def main():
    health = wait_health()
    save("s0_health.json", health)
    record("S0 health", "pass" if health.get("code") == 200 else "fail", **health)
    if health.get("code") != 200:
        save("results.json", results)
        return 1
    if not setup_auth() or not docker_login() or not build_images():
        save("results.json", results)
        return 1

    cmd_app = create_command_app("scenario-auto-2222", C1)
    save("s0_cmd_app.json", cmd_app)
    if not cmd_app:
        record("S0 command app", "fail")
        save("results.json", results)
        return 1
    record("S0 command app", "pass", **cmd_app)
    compose_app = create_compose_app("scenario-compose-2222")
    save("s0_compose_app.json", compose_app)
    record("S0 compose app", "pass" if compose_app else "fail", **(compose_app or {}))

    scenario1(cmd_app)
    if compose_app:
        scenario2(cmd_app, compose_app)
    else:
        record("2 compose app missing", "fail")
    scenario3()

    save("results.json", results)
    save("state.json", {k: v for k, v in state.items() if k != "cli_token"})
    failed = [r for r in results if r["status"] == "fail"]
    print(
        f"SUMMARY pass={sum(1 for r in results if r['status']=='pass')} "
        f"fail={len(failed)} skip={sum(1 for r in results if r['status']=='skip')}",
        flush=True,
    )
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())

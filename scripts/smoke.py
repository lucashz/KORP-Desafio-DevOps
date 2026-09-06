"""Valida o contrato externo, a coleta e o dashboard em uma stack ativa."""
import base64
import datetime
import json
import os
import time
import urllib.error
import urllib.parse
import urllib.request


def get(url, headers=None):
    with urllib.request.urlopen(urllib.request.Request(url, headers=headers or {}), timeout=5) as response:
        return json.load(response)


base = os.environ.get("API_URL", "http://localhost:80")
first = get(base + "/projeto-korp")
time.sleep(0.02)
second = get(base + "/projeto-korp")
assert set(first) == {"nome", "horario"} and first["nome"] == "Projeto Korp", first
assert first["horario"] != second["horario"], "O horário não mudou"
assert second["horario"].endswith("Z"), second
timestamp = datetime.datetime.fromisoformat(second["horario"].replace("Z", "+00:00"))
assert abs((datetime.datetime.now(datetime.timezone.utc) - timestamp).total_seconds()) < 10
try:
    get(base + "/metrics")
    raise AssertionError("Métricas expostas no proxy público")
except urllib.error.HTTPError as error:
    assert error.code == 404, error

query = urllib.parse.urlencode({"query": 'up{job="http-server-projeto-korp"}'})
for attempt in range(20):
    result = get("http://localhost:9090/api/v1/query?" + query)["data"]["result"]
    if result and result[0]["value"][1] == "1":
        break
    time.sleep(2)
else:
    raise AssertionError("Prometheus não está coletando a aplicação")

password = os.environ["GRAFANA_ADMIN_PASSWORD"]
auth = base64.b64encode(("admin:" + password).encode()).decode()
grafana_url = os.environ.get("GRAFANA_URL", "http://localhost:3000")
dashboard = get(grafana_url + "/api/dashboards/uid/projeto-korp", {"Authorization": "Basic " + auth})
assert len(dashboard["dashboard"]["panels"]) >= 6
print("OK: JSON, horário UTC dinâmico, métricas privadas, scrape e dashboard provisionado")

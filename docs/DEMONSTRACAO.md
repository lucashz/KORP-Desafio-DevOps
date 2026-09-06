# Demonstração em videoconferência

## Preparação

1. Disponibilize uma VM Rocky Linux 9/10 ou Ubuntu 24.04 com SSH, sudo, Python 3 e internet. Garanta que a porta 80 esteja livre.
2. Prepare o controlador e inventário conforme README. Verifique conectividade SSH e acesso sudo antes da chamada.
3. Execute o playbook previamente e registre os resultados. Para demonstrar a instalação do zero, restaure um snapshot limpo da VM antes da apresentação; uma segunda execução demonstra idempotência, não instalação inicial.
4. Abra editor com README, Dockerfile, compose, playbook e configuração das métricas. Evite compartilhar `.env`, inventário real ou `.secrets`.
5. Abra duas abas: Grafana e Prometheus. Em servidor remoto, mantenha o túnel SSH ativo.

## Roteiro de 10–15 minutos

**Minutos 0–2 — arquitetura.** Apresente o caminho cliente → NGINX → Go e o caminho Go → Prometheus → Grafana. Mostre que não existe `ports` na API e que existe uma bridge explícita.

**Minutos 2–6 — automação.** Execute:

```bash
ansible-playbook -i ansible/inventory.ini ansible/playbook.yml --ask-become-pass
```

Explique as tarefas durante a execução. O primeiro download pode demorar conforme a rede. Ao final, mostre a resposta JSON e o recap. Execute novamente se houver tempo e compare as mudanças. Não apresente uma execução de sintaxe como provisionamento completo.

**Minutos 6–8 — contrato e isolamento.** No destino, rode:

```bash
cd /opt/projeto-korp
sudo docker compose ps
curl http://localhost:80/projeto-korp
sleep 1
curl http://localhost:80/projeto-korp
sudo docker compose port http-server-projeto-korp 8080
curl -i http://localhost:80/metrics
```

O horário deve mudar e terminar em `Z`; a consulta de porta não deve apresentar publicação; `/metrics` deve retornar 404 pelo proxy. A ausência de porta publicada pode ser mostrada também em `docker compose ps`.

**Minutos 8–11 — monitoramento.** No controlador Linux, execute `sh scripts/load.sh http://IP_DO_SERVIDOR/projeto-korp 300`. Abra o dashboard, ajuste para os últimos quinze minutos e observe taxa, volume e latência. Mostre o target UP no Prometheus.

**Minutos 11–13 — falha e recuperação.** No destino:

```bash
sudo docker compose stop http-server-projeto-korp
```

Observe o proxy falhar, `up` cair após uma coleta e o alerta ficar pending/firing após trinta segundos. Recupere:

```bash
sudo docker compose start http-server-projeto-korp
curl --retry 10 --retry-delay 2 --retry-all-errors --fail http://localhost:80/projeto-korp
```

O DNS do NGINX pode precisar de até dez segundos para renovar o endereço. Mostre a recuperação do gráfico. Explique que o counter reinicia com o processo e que `rate`/`increase` tratam resets.

**Minutos 13–15 — testes e escolhas.** Mostre o CI, testes e limites. Explique uma evolução concreta para produção, como monitorar o caminho inteiro com blackbox exporter e habilitar TLS.

## Diagnóstico rápido

| Sintoma | Verificação |
| --- | --- |
| Porta 80 ocupada | `sudo ss -ltnp 'sport = :80'`; resolva o conflito no host de demonstração |
| NGINX retorna 502 | `sudo docker compose ps` e logs da API/NGINX; confirme healthcheck e DNS |
| Prometheus mostra DOWN | target `http-server-projeto-korp:8080`, rede e logs do serviço |
| Grafana sem dados | datasource, intervalo selecionado, target UP e geração de tráfego |
| Login falha após mudar `.env` | senha inicial já persistida; use a senha original ou procedimento administrativo de reset |
| Ansible sem SSH/sudo | inventário, usuário, chave e `--ask-become-pass` |
| Erro em pacotes conflitantes | use VM limpa ou resolva conscientemente os pacotes da instalação anterior |

## Publicação

Na pasta do projeto, confirme que `.env`, inventário real e `.secrets` são ignorados. Crie um repositório público vazio e conecte o remoto:

```bash
git init -b main
git add .
git diff --cached --stat
git commit -m "Implementa serviço HTTP, monitoramento e provisionamento Ansible"
git remote add origin URL_DO_REPOSITORIO
git push -u origin main
```

Se o repositório já tiver histórico, integre esse histórico antes de publicar; não use force push para contornar divergências. Verifique a execução do workflow no GitHub e abra os links relativos do README após o envio.

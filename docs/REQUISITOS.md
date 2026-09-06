# Rastreabilidade dos requisitos

| Requisito | Implementação | Como verificar |
| --- | --- | --- |
| Serviço Go chamado http-server-projeto-korp | módulo, binário e serviço Compose com esse nome | Dockerfile e compose.yaml |
| Porta interna 8080 | servidor `:8080` | healthcheck e código Go |
| GET /projeto-korp | handler com JSON de dois campos | teste unitário e curl |
| Horário UTC dinâmico | relógio por requisição + UTC/RFC3339Nano | teste com relógio controlado e smoke |
| Build e execução Docker | Dockerfile com dois estágios | compose up --build |
| Docker em Linux | repositório oficial e serviço systemd | execução Ansible no Linux de destino |
| Rede bridge | docker_network com driver bridge | docker network inspect projeto-korp-network |
| API sem porta no host | ausência de ports na API | compose ps/port |
| NGINX oficial em 80:80 | nginx:1.28.0-alpine | Compose e curl |
| Volume /etc/nginx/conf.d | bind mount somente leitura | compose.yaml |
| Arquivo de proxy solicitado | nginx/http-server-projeto-korp.conf | nginx -t |
| Disponibilidade Prometheus | up e /healthz | interromper API e observar coleta |
| Volume de requisições | http_requests_total | tráfego e consultas PromQL |
| Containers Prometheus/Grafana | compose.yaml | compose ps |
| Dashboard | JSON com oito painéis | API do Grafana e interface |
| Um comando Ansible | ansible/playbook.yml | provisionamento completo |
| Validação HTTP e resposta no console | uri, assert e debug | final do playbook |
| Bônus: Grafana automático | datasource, provider e dashboard versionados | primeiro start sem importação manual |
| Repositório público | entrega preparada para GitHub | envio depende da URL do repositório |

Melhorias adicionais: teste de concorrência, race detector, controle de cardinalidade, histogramas, alerta local, usuário não root na API, imagem mínima, timeouts, shutdown gracioso, logs estruturados/rotacionados, healthchecks, senha fora do versionamento, monitoramento restrito a loopback, persistência e CI.

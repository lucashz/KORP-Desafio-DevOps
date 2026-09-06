# Guia técnico para entrevista

## 1. Como apresentar a solução

Explique o caminho de uma requisição: o cliente chega à porta 80 do host, Docker encaminha ao NGINX, o proxy resolve o nome do serviço na rede bridge e acessa a API na porta 8080. A API gera um objeto com nome e horário UTC, serializa em JSON e devolve a resposta pelo mesmo caminho. Separadamente, Prometheus consulta `/metrics` a cada cinco segundos e Grafana consulta o Prometheus para montar os gráficos.

Uma abertura possível, após estudar e executar a entrega: “O projeto é um serviço Go pequeno com infraestrutura reproduzível. A entrada é o NGINX, a aplicação não publica uma porta no host e o monitoramento é configurado por arquivos. O Ansible prepara uma máquina Linux e valida o resultado no final.” Adapte a fala ao que você compreende e consegue demonstrar.

## 2. Serviço Go, do código à resposta

`cmd/server/main.go` concentra o serviço porque o domínio é pequeno. Dividir em muitos pacotes não acrescentaria isolamento útil neste escopo. `newHandler` recebe um relógio e um logger: em produção usa `time.Now`; nos testes usa um horário controlado, permitindo provar que cada requisição consulta o relógio.

`response` contém tags JSON que definem `nome` e `horario`. `UTC()` converte o instante para UTC; `Format(time.RFC3339Nano)` produz uma data com precisão fracionária e sufixo `Z`. O relógio é o do sistema operacional: o serviço não sincroniza a hora; o host deve ter sincronização de horário adequada.

Somente GET é aceito nos endpoints conhecidos. Outros métodos retornam 405 e `Allow: GET`; caminhos desconhecidos retornam 404. O cabeçalho `Content-Type` identifica JSON e `Cache-Control: no-store` evita reuso de uma resposta com horário antigo.

Há timeouts de leitura, escrita e conexão ociosa para impedir que conexões lentas ocupem recursos indefinidamente. `signal.NotifyContext` captura SIGTERM/SIGINT e `Shutdown` dá até dez segundos para requisições em andamento terminarem. Compose permite quinze segundos antes de encerrar à força. Os logs são JSON em stdout, permitindo leitura pelo Docker e futura coleta centralizada.

`healthcheck` é um subcomando do próprio binário. Ele faz GET em `127.0.0.1:8080/healthz` com timeout de dois segundos e retorna código não zero em caso de falha. Isso permite testar uma imagem sem shell nem curl. O endpoint indica que o processo responde; como não há banco de dados, não existe dependência externa a validar.

## 3. Build e isolamento do container

O Dockerfile tem dois estágios. O primeiro contém o compilador Go, baixa dependências a partir de `go.mod`/`go.sum` e compila com `CGO_ENABLED=0`. O segundo usa `scratch` e recebe apenas o executável. `-trimpath` remove caminhos locais do build; `-s -w` reduz símbolos de depuração e tamanho, com a contrapartida de menos informação para debug do binário.

Copiar os arquivos de módulos antes do código permite reaproveitar cache quando só a aplicação muda. `.dockerignore` limita o contexto ao código e módulos; senhas, documentação e inventários não entram na imagem.

O usuário numérico 65532 não é root. A porta 8080 pode ser aberta por usuário comum. O filesystem somente leitura e a remoção de capabilities reduzem permissões. Isso não transforma o container em VM: ele continua compartilhando o kernel do host Linux. Como `scratch` não inclui certificados CA, eventuais chamadas HTTPS externas futuras exigiriam acrescentá-los; a API atual não as faz.

`EXPOSE 8080` documenta a porta interna, mas não publica a porta no host. Quem publica é a propriedade `ports` do Compose. A API não possui essa propriedade.

## 4. Rede e proxy reverso

Uma bridge definida pelo usuário oferece comunicação entre containers e resolução dos nomes dos serviços. `projeto-korp-network` é externa ao Compose para que o Ansible a crie explicitamente. Todos os serviços se conectam a ela.

O NGINX monta `./nginx` em `/etc/nginx/conf.d` como somente leitura. A configuração usa o DNS interno `127.0.0.11` e resolução periódica do backend para tolerar a troca de IP quando a API é recriada. `proxy_pass` mantém o caminho solicitado. Os cabeçalhos encaminhados registram host, IP e protocolo originais; só devem ser usados como identidade após definir uma política de proxies confiáveis.

O bloqueio de `/metrics` evita sua publicação pelo proxy. Prometheus acessa o serviço diretamente na bridge. Grafana e Prometheus escutam no loopback do host para acesso local ou túnel SSH. A porta 80 é pública no host conforme o enunciado. Não há TLS neste exercício.

`depends_on` com `service_healthy` ordena a inicialização, mas não faz gerenciamento contínuo de dependências. `restart: unless-stopped` reinicia processos que encerram inesperadamente; um container marcado unhealthy, mas ainda em execução, não é reiniciado só por isso.

## 5. Métricas e interpretação dos gráficos

**Counter:** `http_requests_total` só cresce durante a vida do processo. Reinicia quando o processo é substituído. `rate(counter[intervalo])` calcula a taxa por segundo e trata resets; `increase(counter[intervalo])` estima o volume no intervalo. A estimativa pode ser fracionária devido à extrapolação entre scrapes, mesmo quando o contador contém inteiros.

Os labels são método, rota e status. A rota desconhecida vira `unmatched`, e métodos incomuns viram `OTHER`, evitando uma série diferente para cada URL ou método arbitrário. IDs de usuário, query strings e caminhos livres não devem virar labels: gerariam cardinalidade alta e consumo de memória.

**Disponibilidade:** `up` é produzido pelo Prometheus, não pelo handler. Vale 1 quando ele coleta o alvo com sucesso e 0 quando a coleta falha. Uma métrica emitida pela própria API não conseguiria emitir zero após o processo morrer. `avg_over_time(up[1h]) * 100` mostra a proporção de coletas bem-sucedidas existentes na última hora. Não cobre intervalos em que o Prometheus esteve desligado e não representa SLA completo: o NGINX pode falhar enquanto a coleta direta continua funcionando. Um blackbox exporter seria uma evolução para medir o caminho do usuário.

**Histogram:** registra duração em buckets, soma e contagem. `histogram_quantile(0.95, ...)` estima o limite abaixo do qual caem 95% das observações, condicionado à resolução dos buckets. A média é `rate(sum) / rate(count)`. Essa latência é a do handler, sem tempo de rede ou NGINX. Sem requisições, média e p95 podem ficar sem valor válido; isso não deve ser convertido artificialmente em zero.

As sondas `/healthz` e `/metrics` ficam fora do contador de tráfego, para que a coleta automática não pareça uso real da API. Requisições inválidas da aplicação entram como 404/405. A implementação não acrescenta recuperação própria de panic nem contabiliza como 500 um panic não tratado; esse seria um aperfeiçoamento se a lógica crescesse.

O alerta `KorpServiceDown` permanece pending durante 30 segundos de falha e depois firing. A regra aparece na interface do Prometheus. Para enviar e-mail ou mensagem seria necessário Alertmanager, roteamento e configuração do canal.

## 6. Grafana automático

`datasources.yml` cria a conexão com `http://prometheus:9090` e UID estável `prometheus`. `dashboards.yml` instrui Grafana a ler os JSONs do diretório montado. O JSON define UID `projeto-korp`, consultas PromQL, painéis e disposição.

Os arquivos são a fonte da configuração; alterações devem ocorrer no repositório. Volumes nomeados preservam usuários e dados do Grafana e séries do Prometheus entre recriações. Montagens `ro` protegem as configurações contra escrita pelos containers, enquanto volumes de dados permanecem graváveis.

A senha é gerada uma vez pelo Ansible e guardada fora do versionamento. Ela não aparece no console. A variável de ambiente inicializa o usuário quando o banco do Grafana está vazio; mudar só essa variável não altera a senha de um usuário já criado.

## 7. Ansible e idempotência

Inventário descreve destinos e acesso. Playbook descreve o estado desejado. `become: true` eleva permissões para instalar pacotes e administrar Docker. Facts identificam distribuição Linux, versão e arquitetura.

O fluxo é: validar sistema e conflitos, instalar certificados, adicionar chave/repositório, instalar pacotes, iniciar serviço, copiar deploy, gerar senha, criar rede bridge, construir/iniciar stack, validar contrato e monitoramento, exibir resposta.

Foram usados módulos de pacote, arquivo, serviço, rede, Compose e HTTP em vez de uma sequência opaca de comandos shell. Eles detectam diferenças e facilitam reexecução. `register` guarda o resultado da cópia: se os arquivos mudarem, há build e recriação para que NGINX/Prometheus releiam configurações. Sem mudanças, usa a política normal de build/recriação. Essa opção recria todos os serviços quando qualquer arquivo de deploy muda; é simples, mas causa uma breve interrupção. Uma evolução seria separar notificações por componente.

O playbook não remove Docker de outra origem silenciosamente: identifica pacotes incompatíveis e explica a necessidade de um host limpo. Não substitui a provisão da VM, o usuário SSH ou a instalação do próprio Ansible no controlador. “Um comando” aplica-se ao provisionamento do destino depois desses pré-requisitos.

Idempotência precisa ser verificada com uma segunda execução real e observação de `changed`, não apenas afirmada. Atualização do cache APT, mudança de chave remota e alterações externas podem afetar o resultado. As distribuições previstas são Rocky Linux 9/10 e Ubuntu 24.04; a compatibilidade deve ser confirmada pela execução do playbook.

## 8. Testes e limitações

Os testes unitários verificam contrato, relógio dinâmico em UTC, métodos, caminhos, labels, exclusão das sondas e cem requisições concorrentes. `-race` procura acessos concorrentes inseguros. `go vet` encontra padrões suspeitos; não substitui testes. O teste integrado usa a stack real e verifica o acesso pelo proxy, coleta e dashboard via API autenticada.

O CI faz também syntax-check Ansible, mas isso não instala Docker em uma VM. Diferencie teste de sintaxe, teste dos containers e execução completa do provisionamento ao apresentar as evidências. Consulte `VALIDACAO.md` para saber o que foi efetivamente executado.

Não há banco de negócio, fila, Kubernetes ou camadas de persistência desnecessárias. O projeto privilegia demonstrar os requisitos com componentes que possam ser explicados e operados. As limitações conhecidas são host único, HTTP, falta de notificações externas, falta de backup automático e disponibilidade medida por scrape.

## 9. Perguntas para praticar sem consultar o código

1. Por que a API usa 8080 e o cliente usa 80?
2. Qual a diferença entre `EXPOSE` e `ports`?
3. O que ocorre se a API reiniciar e ganhar outro IP?
4. Por que usar `rate` em um counter?
5. `up == 1` garante que o cliente acessa o NGINX?
6. Por que excluir `/metrics` do contador?
7. O que é cardinalidade e por que agrupar 404?
8. Como Grafana descobre o dashboard sem cliques?
9. O que persiste após `docker compose down`?
10. O que muda na segunda execução do Ansible?
11. Por que uma alteração no `.env` pode não trocar a senha do Grafana?
12. Como diagnosticar 502: processo, rede, DNS ou proxy?
13. Quais testes passaram de fato e quais dependem da VM?
14. Quais seriam suas três primeiras melhorias para produção?

Se algum ponto não estiver claro, execute um experimento pequeno: pare a API, observe `up`, tente acessar o proxy e compare os logs. Isso ajuda a construir uma explicação baseada em comportamento observado.

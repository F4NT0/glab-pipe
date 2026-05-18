
# Refatoração do projeto glab-pipeline


## Antes de começar

1. Leia todo o documento e me pergunte se tem alguma duvida.
2. Crie um arquivo de configuração para colocar os projetos que passei, esse arquivo deve ser lido pelo sistema onde pode ser adicionado mais projetos.
3. Valide um possivel consumo de tokens por esse projeto.
4. Não precisa perguntar se desejo rodar algum comando, pode fazer todos os comandos necessários para validar e implementar.
5. No final, me mostre arquivos modificados, processos feitos, validações para garantir funcionamento
6. Certifique de criar um .exe para o instalador e para o programa e me mostre o comando para eu poder debugar manualmente se quiser
7. Crie/Atualize o README.md do projeto com todos os detalhes técnicos e de uso do software, mostrando o passo a passo de uso e deixando espaços para eu colocar imagens de exemplo.

## Busca de projetos

1. Adicione os seguintes projetos no programa:

- dfs/support/dfs-case-management/casemanagement/account-processor-api
- dfs/support/dfs-case-management/casemanagement/case-connector-api
- dfs/support/dfs-case-management/casemanagement/case-gateway
- dfs/support/dfs-case-management/casemanagement/case-receiver-api

## Processo 

1. Devo iniciar o programa com o comando glab-pipe no terminal (esse deve ser o comando que o instalador deve configurar no meu terminal para eu iniciar o projeto).
2. Mostra o menu inicial com o meu ascii text e a opção de selecionar qual repositório desejo verificar
3. O programa deve mostrar uma tabela com todas as ultimas 10 pipelines que foram rodadas no projeto onde na tabela deve ser o status da pipeline como:

```
 (\uf192) em azul dizendo que está running
 (\uf05d) em verde dizendo que foi success
 (\uf52f) em vermelho dizendo que foi failed
 (\ueabd) em cinza dizendo que foi canceled
 (\uf2be) em amarelo dizendo que foi manual
```

Nessa tabela a primeira coluna se chama Status com esses icones, na segunda coluna é o ID da pipeline (ex. #56012567), a terceira coluna é o nome da branch que pertence essa pipeline (ex. story/CUC-689) e a quarta coluna é quando a pipeline foi iniciada.

Obs: se consegue essas informações usando o comando do `glab ci list` com o projeto que foi selecionado.

4. Devo poder selecionar qual pipeline desejo verificar, onde quando clicar em Enter devo ser enviado a outra visão do sistema.
5. Nessa outra visão, deve mostrar os jobs da pipeline, em uma lista também, onde tem os seguintes detalhes

```
 (\uf192) em azul dizendo que está running
 (\uf05d) em verde dizendo que foi success
 (\uf52f) em vermelho dizendo que foi failed
 (\ueabd) em cinza dizendo que foi canceled
 (\uf2be) em amarelo dizendo que foi manual
 (\uf01d) em amarelo dizendo que precisa rodar manualmente
```

Nessa tabela de jobs, a primeira coluna é o status usando os icones acima, a segunda coluna é o nome do job.

Obs: se consegue essas informações usando o comando `glab ci get --pipeline-id=PipelineId`

Acima dessa tabela de jobs deve ter as informações principais da pipeline

- ID da pipeline
- Status da pipeline
- Source da Pipeline
- Nome da branco que foi rodada (ref)
- User que rodou a pipeline
- Quando a pipeline foi criada
- Quando a pipeline foi atualizada

Nessa página, diferente da página de todas as pipelines, deve ficar se atualizando a cada 2 segundos, mostrando atualização da pipeline e dos jobs.

6. Devo poder selecionar qual job abrir dessa tabela, onde posso selecionar qualquer um dos jobs.
7. Quando eu clicar no Job, ele deve abrir um modal quase do tamanho total do software aberto (o TUI de todo o software deve alterar de tamanho devido ao tamanho do terminal aberto, o modal deve ser quase o tamanho todo, para poder ver bem os logs).
8. Nesse modal deve mostrar os logs do job, pode ser pego usando o comando `glab ci trace nome-job --pipeline-id=PipelineId`.
9. Caso o job ainda esteja rodando, deve ser atualizado a casa 2 segundos o modal para poder ver sendo atualizado o status do job e do log, o modal deve ter no título dele o status do job usando os icones mencionados, o nome do job, código da pipeline e branch da pipeline para poder ter o trace completo do processo. 
10. Posso fechar o modal com Esc para voltar a lista de jobs que deve continuar se atualizando enquanto tiver um job como running, quando a pipeline estiver concluida ele para de ficar atualizando o modal do job e a página dos jobs.
11. Posso voltar com Esc na página dos jobs para a página inicial com todas as pipelines do projeto, onde se tiver com alguma pipeline rodando deve ficar se atualizando a cada 2 segundos para ver o status das pipelines, se não tiver nenhum rodando não precisa ficar atualizando.
12. Se eu clicar em Esc na tela das pipelines deve me levar de volta ao menu principal, onde posso clicar em Enter de novo para escolher outro projeto.

## instalador

1. O instalador deve ser utilizando um TUI parecido com o do projeto C:\Users\Gabriel_Stundner\source\repos\GITHUB\clidocs\install.exe
2. Deve ser verificado se possui o glab instalado e configurado com o gitlab.example.com, deve verificar se tem uma fonte nerd font no terminal (mostra um warning explicando que os icones não aparecem corretamente sem uma fonte nerd font) e se está usando o powershell (mostra um warning dizendo que é melhor usar o powershell).
3. Instala o .exe desse programa no computador em um lugar onde o comando `glab-pipe` encontre o programa.
4. Coloca no PATH para que tanto o powershell quanto o command prompt consiga rodar o comando glab-pipe.
5. O instalador deve ter todo o visual TUI seguindo o exemplo do clidocs.

## Utilização do comando

1. Caso somente coloque glab-pipe no terminal ele abre o menu principal para selecionar o projeto.
2. Caso rode o comando como `glab-pipe . ` ele deve verificar se o projeto atual se encontra no gitlab.example.com e procura o caminho correto dele, se o caminho já existir nas configurações (ele está salvo como os que pedi acima) senão ele deve procurar se for a primeira vez que está sendo aberto.
3. Achando o projeto pelo glab, ele envia para a tela das pipelines do projeto e segue o processo já explicado.
4. Caso rode o comando `glab-pipe --source <caminho>` ele já irá pegar a localização do projeto e se o usuário tiver permissão de visualizar ele abre a tela das pipelines, senão apresenta uma mensagem na tela dizendo que não encontrou o programa.




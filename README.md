This program has three hardcoded nodes with the following IDs: 1, 2, 3 \
Each node has a hardcoded port it's listening on, and the other nodes also know which ports the others listen on

To start the program you can navigate to the client folder \
then from the terminal write `go run client.go --id='<id>'`
where \<id> is either 1, 2 or 3

You can the manually request access to the Critical Section by writing anything in the terminal
(or even nothing) followed by `enter`

This then activates the Ricart-Agrawala algorithm, and in the log
you can see all the steps

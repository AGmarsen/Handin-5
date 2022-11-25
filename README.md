# Handin-5

## Starting the system

Open 3 terminals, one for each node.

In each terminal run the following command: go run \<path to node.go\> \<x\>

Replace \<x\> with the number 0 in the first terminal. 1 in the second and 2 in the third.

Afterwards open a terminal for each bidder/client that can bid on the auction.

In each terminal run the command: go run \<path to bidder.go\> \<name\>

\<name\> is a string that represents the name of the client. You get to pick.

## Using the program

Now that the system is running you can simulate a crash by using ctrl + C or by typing "end" on a node terminal.

In the bidder terminals you can bid by typing "bid x" (replace x with a number) and see the auction status by typing "status".

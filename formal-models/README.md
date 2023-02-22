## dBFT formal models

This section contains a set of dBFT's formal specifications written in
[TLA⁺](https://lamport.azurewebsites.net/tla/tla.html) language. The models
describe the core algorithm logic represented in a high-level way and can be used
to illustrate some basic dBFT concepts and to validate the algorithm in terms of
liveness and fairness. It should be noted that presented models do not precisely
follow the dBFT implementation presented in the repository and may omit some
implementation details in favor of the specification simplicity and the
fundamental philosophy of the TLA⁺. However, the presented models directly
reflect some liveness problems dBFT 2.0 has; the models can and are aimed to be
used for the dBFT 2.0 liveness evaluation and further algorithm improvements.

Any contributions, questions and discussions on the presented models are highly
appreciated.

## dBFT 2.0 models

### Basic dBFT 2.0 model

This specification is a basis that was taken for the further algorithm
investigation. We recommend to begin acquaintance with the dBFT models from this
one.
 
The specification describes the process of a single block acceptance: the set of
resource managers `RM` (which is effectively a set of consensus nodes)
communicating via the shared consensus message pool `msgs` and taking the
decision in a few consensus rounds (views). Each consensus node has its own state
at each step of the behaviour. Consensus node may send a consensus message by
adding it to the `msgs` pool. To perform the transition between states the
consensus node must send a consensus message or there must be a particular set of
consensus messages in the shared message pool required for a particular
transition.

Here's the scheme of transitions between consensus node states:

![Basic dBFT model transitions scheme](./.github/dbft.png)

The specification also describes two kinds of malicious nodes behaviour that can
be combined, i.e. enabled or disabled independently for each particular node:

1. "Dead" nodes. "Dead" node is completely excluded from the consensus process
   and not able to send the consensus messages and to perform state transitions.
   The node may become "dead" at any step in the middle of the consensus process.
   Once the node becomes "dead" there's no way for it to rejoin the consensus
   process.
2. "Faulty" nodes. "Faulty" node is allowed to send consensus messages of *any*
   type at *any* step and to change its view without regarding the dBFT view
   changing rules. The node may become "faulty" at any step in the middle of the
   consensus process. Once the node becomes "faulty" there's no way for it to
   become "good" again.

The specification contains several invariants and liveness properties that must
be checked by the TLC Model Checker. These formulas mostly describe two basic
concepts that dBFT algorithm expected to guarantee:

1. No fork must happen. There must be no situation such that two different
   blocks are accepted at two different consensus rounds (views).
2. The block must always be accepted. There must be no situation such that nodes
   are stuck in the middle of consensus process and can't take any further steps.

The specification is written and working under several assumptions:

1. All consensus messages are valid. In real life it is guaranteed by verifiable
   message signatures. In case if malicious or corrupted message is received it
   won't be handled by the node.
2. The exact timeouts (e.g. t/o on waiting a particular consensus message, etc.)
   are not included into the model. However, the model covers timeouts in
   general, i.e. the timeout is just the possibility to perform a particular
   state transition.
3. All consensus messages must eventually be delivered to all nodes, but the
   exact order of delivering isn't guaranteed.
4. The maximum number of consensus rounds (views) is restricted. This constraint
   was introduced to reduce the number of possible model states to be checked.
   The threshold may be specified via model configuration, and it is highly
   recommended to keep this setting less or equal to the number of consensus
   nodes.

Here you can find the specification file and the TLC Model Checker launch
configuration:

* [TLA⁺ specification](./dbft/dbft.tla)
* [TLC Model Checker configuration](./dbft/dbft___AllGoodModel.launch)

### Extended dBFT 2.0 model

This is an experimental dBFT 2.0 specification that extends the
[basic model](#basic-dbft-20-model) in the following way: besides the shared pool
of consensus messages `msgs` each consensus node has its own local pool of
received and handled messages. Decisions on transmission between the node states
are taken by the node based on the state of the local message pool. This approach
allows to create more accurate low-leveled model which is extremely close to the
dBFT implementation presented in this repository. At the same time such approach
*significantly* increases the number of considered model states which leads to
abnormally long TLC Model Checker runs. Thus, we do not recommend to use this
model in development and place it here as an example of alternative (and more
detailed) dBFT specification. These two models are expected to be equivalent in
terms of the liveness locks that can be discovered by both of them, and, speaking
the TLA⁺ language, the Extended dBFT specification implements the
[basic one](#basic-dbft-20-model) (which can be proven and written in TLA⁺, but
stays out of the task scope).

Except for this remark and a couple of minor differences all the
[basic model](#basic-dbft-20-model) description, constraints and assumptions are
valid for the Extended specification as far. Thus, we highly recommend to
consider the [basic model](#basic-dbft-20-model) before going to the Extended
one.
 
Here you can find the specification file and the TLC Model Checker launch
configuration:

* [TLA⁺ specification](./dbftMultipool/dbftMultipool.tla)
* [TLC Model Checker configuration](./dbftMultipool/dbftMultipool___AllGoodModel.launch)

## How to run/check the TLA⁺ specification

### Prerequirements

1. Download and install the TLA⁺ Toolbox following the
   [official guide](http://lamport.azurewebsites.net/tla/toolbox.html).
2. Read the brief introduction to the TLA⁺ language and TLC Model Checker at the
   [official site](http://lamport.azurewebsites.net/tla/high-level-view.html).
3. Download and take a look at the
   [TLA⁺ cheat sheet](https://lamport.azurewebsites.net/tla/summary-standalone.pdf).
4. For a proficient learning watch the
   [TLA⁺ Video Course](https://lamport.azurewebsites.net/video/videos.html) and
   read the [Specifying Systems book](http://lamport.azurewebsites.net/tla/book.html?back-link=tools.html#documentation).

### Running the TLC model checker

1. Clone the [repository](https://github.com/nspcc-dev/dbft.git).
2. Open the TLA⁺ Toolbox, open new specification and provide path to the desired
   `*.tla` file that contains the specification description.
3. Create the model named `AllGoodModel` in the TLA⁺ Toolbox.
4. Copy the corresponding `*___AllGoodModel.launch` file to the `*.toolbox`
   folder. Reload/refresh the model in the TLA⁺ Toolbox.
5. Open the `Model Overview` window in the TLA⁺ Toolbox  and check that behaviour
   specification, declared constants, invariants and properties of the model are
   filled in with some values.
6. Press `Run TLC on the model` bottom to start the model checking process and
   explore the progress in the `Model Checkng Results` window.
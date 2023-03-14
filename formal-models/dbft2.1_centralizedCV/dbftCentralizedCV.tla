-------------------------------- MODULE dbftCentralizedCV --------------------------------

EXTENDS
  Integers,
  FiniteSets
  
CONSTANTS
  \* RM is the set of consensus node indexes starting from 0.
  \* Example: {0, 1, 2, 3}
  RM,

  \* RMFault is a set of consensus node indexes that are allowed to become
  \* FAULT in the middle of every considered behavior and to send any
  \* consensus message afterwards. RMFault must be a subset of RM. An empty
  \* set means that all nodes are good in every possible behaviour.
  \* Examples: {0}
  \*           {1, 3}
  \*           {}
  RMFault,

  \* RMDead is a set of consensus node indexes that are allowed to die in the
  \* middle of every behaviour and do not send any message afterwards. RMDead
  \* must be a subset of RM. An empty set means that all nodes are alive and
  \* responding in in every possible behaviour. RMDead may intersect the
  \* RMFault set which means that node which is in both RMDead and RMFault
  \* may become FAULT and send any message starting from some step of the
  \* particular behaviour and may also die in the same behaviour which will
  \* prevent it from sending any message.
  \* Examples: {0}
  \*           {3, 2}
  \*           {}
  RMDead,

  \* MaxView is the maximum allowed view to be considered (starting from 0,
  \* including the MaxView itself). This constraint was introduced to reduce
  \* the number of possible model states to be checked. It is recommended to
  \* keep this setting not too high (< N is highly recommended).
  \* Example: 2
  MaxView

VARIABLES
  \* rmState is a set of consensus node states. It is represented by the
  \* mapping (function) with domain RM and range RMStates. I.e. rmState[r] is
  \* the state of the r-th consensus node at the current step.
  rmState,

 \* msgs is the shared pool of messages sent to the network by consensus nodes.
 \* It is represented by a subset of Messages set.
  msgs,
  
  \* blockAccepted holds the view number when the accepted block's proposal
  \* was firstly created for each consensus node. It is represented by a
  \* mapping RM -> Nat and needed for InvTwoBlocksAcceptedAdvanced invariant
  \* evaluation.
  blockAccepted

\* vars is a tuple of all variables used in the specification. It is needed to
\* simplify fairness conditions definition.
vars == <<rmState, msgs, blockAccepted>>

\* N is the number of validators.
N == Cardinality(RM)

\* F is the number of validators that are allowed to be malicious.
F == (N - 1) \div 3

\* M is the number of validators that must function correctly.
M == N - F

\* These assumptions are checked by the TLC model checker once at the start of
\* the model checking process. All the input data (declared constants) specified
\* in the "Model Overview" section must satisfy these constraints.
ASSUME
  /\ RM \subseteq Nat
  /\ N >= 4
  /\ 0 \in RM
  /\ RMFault \subseteq RM
  /\ RMDead \subseteq RM
  /\ Cardinality(RMFault) <= F
  /\ Cardinality(RMDead) <= F
  /\ Cardinality(RMFault \cup RMDead) <= F
  /\ MaxView \in Nat
  /\ MaxView <= 2

\* RMStates is a set of records where each record holds the node state and
\* the node current view.
RMStates == [
              type: {"initialized", "prepareSent", "commitSent", "blockAccepted",  "cv1", "cv2", "bad", "dead"},
              view : Nat
            ]
\* Messages is a set of records where each record holds the message type,
\* the message sender, sender's view by the moment when message was sent
\* (view field), sender's view by the first moment the PrepareRequest was
\* originally proposed (sourceView field) and the target view for
\* ChangeView[1,2]/DoChangeView[1,2] messages (targetView field).
Messages == [type : {"PrepareRequest", "PrepareResponse", "Commit", "ChangeView1", "ChangeView2", "DoChangeView1", "DoChangeView2"}, rm : RM, view : Nat, targetView : Nat, sourceView : Nat]

\* -------------- Useful operators --------------

\* IsPrimaryInTheView is an operator defining whether provided node r
\* is primary in the consensus round view from the r's point of view.
\* It is a mapping from RM to the set of {TRUE, FALSE}.
IsPrimaryInTheView(r, view) == view % N = r

\* IsPrimary is an operator defining whether provided node r is primary
\* for the current round from the r's point of view. It is a mapping
\* from RM to the set of {TRUE, FALSE}.
IsPrimary(r) == IsPrimaryInTheView(r, rmState[r].view)

\* GetPrimary is an operator defining mapping from round index to the RM that
\* is primary in this round.
GetPrimary(view) == CHOOSE r \in RM : view % N = r

\* GetNewView returns new view number based on the previous node view value.
\* Current specifications only allows to increment view.
GetNewView(oldView) == oldView + 1

\* PrepareRequestSentOrReceived denotes whether there's a PrepareRequest
\* message received from the current round's speaker (as the node r sees it).
PrepareRequestSentOrReceived(r) == Cardinality({msg \in msgs : msg.type = "PrepareRequest"/\ msg.rm = GetPrimary(rmState[r].view) /\ msg.view = rmState[r].view}) /= 0

\* -------------- Safety temporal formula --------------

\* Init is the initial predicate initializing values at the start of every
\* behaviour.
Init ==
  /\ rmState = [r \in RM |-> [type |-> "initialized", view |-> 0]]
  /\ msgs = {}
  /\ blockAccepted = [r \in RM |-> 0]

\* RMSendPrepareRequest describes the primary node r originally broadcasting PrepareRequest.
RMSendPrepareRequest(r) ==
  /\ rmState[r].type = "initialized"
  /\ IsPrimary(r)
  /\ rmState' = [rmState EXCEPT ![r].type = "prepareSent"]
  /\ msgs' = msgs \cup {[type |-> "PrepareRequest", rm |-> r, view |-> rmState[r].view, targetView |-> 0, sourceView |-> rmState[r].view]}
  /\ UNCHANGED <<blockAccepted>>

\* RMSendPrepareResponse describes non-primary node r receiving PrepareRequest from
\* the primary node of the current round (view) and broadcasting PrepareResponse.
\* This step assumes that PrepareRequest always contains valid transactions and
\* signatures.
RMSendPrepareResponse(r) ==
  /\ rmState[r].type = "initialized"
  /\ \neg IsPrimary(r)
  /\ PrepareRequestSentOrReceived(r)
  /\ LET pReq == CHOOSE msg \in msgs : msg.type = "PrepareRequest" /\ msg.rm = GetPrimary(rmState[r].view) /\ msg.view = rmState[r].view
     IN /\ rmState' = [rmState EXCEPT ![r].type = "prepareSent"]
        /\ msgs' = msgs \cup {[type |-> "PrepareResponse", rm |-> r, view |-> rmState[r].view, targetView |-> 0, sourceView |-> pReq.sourceView]}
        /\ UNCHANGED <<blockAccepted>>


\* RMSendCommit describes node r sending Commit if there's enough PrepareResponse
\* messages.
RMSendCommit(r) ==
  /\ rmState[r].type = "prepareSent"
  /\ Cardinality({
                   msg \in msgs : (msg.type = "PrepareResponse" \/ msg.type = "PrepareRequest") /\ msg.view = rmState[r].view
                }) >= M 
  /\ PrepareRequestSentOrReceived(r)
  /\ LET pReq == CHOOSE msg \in msgs : msg.type = "PrepareRequest" /\ msg.rm = GetPrimary(rmState[r].view) /\ msg.view = rmState[r].view
     IN /\ rmState' = [rmState EXCEPT ![r].type = "commitSent"]
        /\ msgs' = msgs \cup {[type |-> "Commit", rm |-> r, view |-> rmState[r].view, targetView |-> 0, sourceView |-> pReq.sourceView]}
        /\ UNCHANGED <<blockAccepted>>

\* RMAcceptBlock describes node r collecting enough Commit messages and accepting
\* the block.
RMAcceptBlock(r) ==
  /\ rmState[r].type /= "bad"
  /\ rmState[r].type /= "dead"
  /\ rmState[r].type /= "blockAccepted"
  /\ PrepareRequestSentOrReceived(r)
  /\ Cardinality({msg \in msgs : msg.type = "Commit" /\ msg.view = rmState[r].view}) >= M
  /\ LET pReq == CHOOSE msg \in msgs : msg.type = "PrepareRequest" /\ msg.rm = GetPrimary(rmState[r].view) /\ msg.view = rmState[r].view
     IN /\ rmState' = [rmState EXCEPT ![r].type = "blockAccepted"]
        /\ blockAccepted' = [blockAccepted EXCEPT ![r] = pReq.sourceView]
        /\ UNCHANGED <<msgs>>

\* FetchBlock describes node r that fetches the accepted block from some other node.
RMFetchBlock(r) ==
  /\ rmState[r].type /= "bad"
  /\ rmState[r].type /= "dead"
  /\ rmState[r].type /= "blockAccepted"
  /\ \E rmAccepted \in RM : /\ rmState[rmAccepted].type = "blockAccepted"
                            /\ rmState' = [rmState EXCEPT ![r].type = "blockAccepted", ![r].view = rmState[rmAccepted].view]
                            /\ blockAccepted' = [blockAccepted EXCEPT ![r] = blockAccepted[rmAccepted]]
                            /\ UNCHANGED <<msgs>>

\* RMSendChangeView1 describes node r sending ChangeView1 message on timeout
\* during waiting for PrepareResponse or node r in "cv2" state receiving M
\* messages from the I stage not more than F of them are preparations.
RMSendChangeView1(r) ==
  /\ \/ /\ rmState[r].type = "initialized"
        /\ \neg IsPrimary(r)
     \/ /\ rmState[r].type = "cv2"
        /\ Cardinality({msg \in msgs : /\ (msg.type = "ChangeView1" \/ msg.type = "PrepareRequest" \/ msg.type = "PrepareResponse")
                                      /\ msg.view = rmState[r].view
                     }) >= M
        /\ Cardinality({msg \in msgs : /\ (msg.type = "PrepareRequest" \/ msg.type = "PrepareResponse")
                                       /\ msg.view = rmState[r].view
                      }) <= F
  /\ rmState' = [rmState EXCEPT ![r].type = "cv1"]
  /\ msgs' = msgs \cup {[type |-> "ChangeView1", rm |-> r, view |-> rmState[r].view, targetView |-> GetNewView(rmState[r].view), sourceView |-> rmState[r].view]}
  /\ UNCHANGED <<blockAccepted>>

\* RMSendChangeView1FromCV1 describes node r sending ChangeView1 message on timeout
\* during waiting for DoCV1 signal from the next primary.
RMSendChangeView1FromCV1(r) ==
  /\ rmState[r].type = "cv1"
  /\ LET cv1s == {msg \in msgs : msg.type = "ChangeView1" /\ msg.view = rmState[r].view}
         myCV1s == {msg \in cv1s : msg.rm = r}
         myBestCV1 == CHOOSE msg \in myCV1s : \A other \in myCV1s : msg.targetView >= other.targetView
     IN /\ Cardinality({msg \in cv1s : msg.targetView = myBestCV1.targetView}) >= M
        /\ \neg IsPrimaryInTheView(r, myBestCV1.targetView)
        /\ msgs' = msgs \cup {[type |-> "ChangeView1", rm |-> r, view |-> rmState[r].view, targetView |-> GetNewView(myBestCV1.targetView), sourceView |-> rmState[r].view]}
        /\ UNCHANGED <<rmState, blockAccepted>>

\* RMSendChangeView2 describes node r sending ChangeView message on timeout
\* during waiting for enough Prepare messages OR from CV1 after not receiving
\* enough ChangeView1 messages.
RMSendChangeView2(r) ==
  /\ \/ rmState[r].type = "prepareSent"
     \/ rmState[r].type = "commitSent"
     \/ /\ rmState[r].type = "cv1"
        /\ Cardinality({msg \in msgs : /\ (msg.type = "PrepareRequest" \/ msg.type = "PrepareResponse" \/ msg.type = "ChangeView1")
                                       /\ msg.view = rmState[r].view 
                                       /\ (msg.targetView = 0 \/ msg.targetView = GetNewView(rmState[r].view))
                      }) >= M
        /\ Cardinality({msg \in msgs : /\ (msg.type = "PrepareRequest" \/ msg.type = "PrepareResponse")
                                       /\ msg.view = rmState[r].view
                                       /\ (msg.targetView = 0 \/ msg.targetView = GetNewView(rmState[r].view))
                      }) > F
  /\ LET pReq == CHOOSE msg \in msgs : msg.type = "PrepareRequest" /\ msg.rm = GetPrimary(rmState[r].view) /\ msg.view = rmState[r].view
     IN /\ rmState' = [rmState EXCEPT ![r].type = "cv2"]
        /\ msgs' = msgs \cup {[type |-> "ChangeView2", rm |-> r, view |-> rmState[r].view, targetView |-> GetNewView(rmState[r].view), sourceView |-> pReq.sourceView]}
        /\ UNCHANGED <<blockAccepted>>

\* RMSendChangeView2FromCV2 describes node r sending ChangeView2 message on timeout
\* during waiting for DoCV2 signal from the next primary.
RMSendChangeView2FromCV2(r) ==
  /\ rmState[r].type = "cv2"
  /\ LET cv2s == {msg \in msgs : msg.type = "ChangeView2" /\ msg.view = rmState[r].view}
         myCV2s == {msg \in cv2s : msg.rm = r}
         myBestCV2 == CHOOSE msg \in myCV2s : \A other \in myCV2s : msg.targetView >= other.targetView
         pReq == CHOOSE msg \in msgs : msg.type = "PrepareRequest" /\ msg.rm = GetPrimary(rmState[r].view) /\ msg.view = rmState[r].view
     IN /\ Cardinality({msg \in cv2s : msg.targetView = myBestCV2.targetView}) >= M
        /\ \neg IsPrimaryInTheView(r, myBestCV2.targetView)
        /\ msgs' = msgs \cup {[type |-> "ChangeView2", rm |-> r, view |-> rmState[r].view, targetView |-> GetNewView(myBestCV2.targetView), sourceView |-> pReq.sourceView]}
        /\ UNCHANGED <<rmState, blockAccepted>>

\* RMSendDoCV1ByLeader describes node r that collects enough ChangeView1 messages
\* with target view such that the node r is leader in this view. The leader r
\* broadcasts DoChangeView1 message and the newly-created PrepareRequest message
\* for this view.
RMSendDoCV1ByLeader(r) ==
  /\ rmState[r].type /= "bad"
  /\ rmState[r].type /= "dead"
  /\ rmState[r].type /= "blockAccepted"
  /\ LET cv1s == {msg \in msgs : msg.type = "ChangeView1" /\ msg.view = rmState[r].view}
         followersCV1s == {msg \in cv1s : IsPrimaryInTheView(r, msg.targetView)} \* TODO: this condition won't work starting from N+1 view!
         targetView == CHOOSE x \in 1..(MaxView+1) : IsPrimaryInTheView(r, x)
     IN /\ Cardinality(followersCV1s) >= M
        /\ rmState' = [rmState EXCEPT ![r].type = "prepareSent", ![r].view = targetView]
        /\ msgs' = msgs \cup {[type |-> "DoChangeView1", rm |-> r, view |-> rmState[r].view, targetView |-> targetView, sourceView |-> targetView], [type |-> "PrepareRequest", rm |-> r, view |-> targetView, targetView |-> 0, sourceView |-> targetView]}
        /\ UNCHANGED <<blockAccepted>>

\* RMSendDoCV2ByLeader describes node r that collects enough ChangeView2 messages
\* with target view such that the node r is leader in this view. The leader r
\* broadcasts DoChangeView2 message and the old PrepareRequest message that
\* was migrated from the previous view without changes.
RMSendDoCV2ByLeader(r) ==
  /\ rmState[r].type /= "bad"
  /\ rmState[r].type /= "dead"
  /\ rmState[r].type /= "blockAccepted"
  /\ LET cv2s == {msg \in msgs : msg.type = "ChangeView2" /\ msg.view = rmState[r].view}
         followersCV2s == {msg \in cv2s : IsPrimaryInTheView(r, msg.targetView)}
         targetView == CHOOSE x \in 1..(MaxView+1) : IsPrimaryInTheView(r, x)
     IN /\ Cardinality(followersCV2s) >= M
        /\ LET pReq == CHOOSE msg \in msgs : msg.type = "PrepareRequest" /\ msg.rm = GetPrimary(rmState[r].view) /\ msg.view = rmState[r].view
           IN /\ rmState' = [rmState EXCEPT ![r].type = "prepareSent", ![r].view = targetView]
              /\ msgs' = msgs \cup {[type |-> "DoChangeView2", rm |-> r, view |-> rmState[r].view, targetView |-> targetView, sourceView |-> pReq.sourceView], [type |-> "PrepareRequest", rm |-> r, view |-> targetView, targetView |-> 0, sourceView |-> pReq.sourceView]}
              /\ UNCHANGED <<blockAccepted>>

\* RMReceiveDoCV1FromLeader descibes node r that receives DoChangeView1 message from
\* the leader of the target DoCV1's view and changes its view.
RMReceiveDoCV1FromLeader(r) ==
  /\ rmState[r].type /= "bad"
  /\ rmState[r].type /= "blockAccepted"
  /\ rmState[r].type /= "dead"
  /\ Cardinality({msg \in msgs : msg.type = "DoChangeView1" /\ msg.targetView > rmState[r].view}) /= 0
  /\ LET doCV1s == {msg \in msgs : msg.type = "DoChangeView1" /\ msg.targetView > rmState[r].view}
         latestDoCV1 == CHOOSE msg \in doCV1s : \A other \in doCV1s : msg.targetView >= other.targetView
     IN /\ rmState' = [rmState EXCEPT ![r].type = "initialized", ![r].view = latestDoCV1.targetView]
        /\ UNCHANGED <<msgs, blockAccepted>>

\* RMReceiveDoCV2FromLeader descibes node r that receives DoChangeView2 message from
\* the leader of the target DoCV2's view and changes its view.
RMReceiveDoCV2FromLeader(r) ==
  /\ rmState[r].type /= "bad"
  /\ rmState[r].type /= "blockAccepted"
  /\ rmState[r].type /= "dead"
  /\ Cardinality({msg \in msgs : msg.type = "DoChangeView2" /\ msg.targetView > rmState[r].view}) /= 0
  /\ LET doCV2s == {msg \in msgs : msg.type = "DoChangeView2" /\ msg.targetView > rmState[r].view}
         latestDoCV2 == CHOOSE msg \in doCV2s : \A other \in doCV2s : msg.targetView >= other.targetView
     IN /\ rmState' = [rmState EXCEPT ![r].type = "initialized", ![r].view = latestDoCV2.targetView]
        /\ UNCHANGED <<msgs, blockAccepted>>

\* RMBeBad describes the faulty node r that will send any kind of consensus message starting
\* from the step it's gone wild. This step is enabled only when RMFault is non-empty set.
RMBeBad(r) ==
  /\ r \in RMFault
  /\ Cardinality({rm \in RM : rmState[rm].type = "bad"}) < F
  /\ rmState' = [rmState EXCEPT ![r].type = "bad"]
  /\ UNCHANGED <<msgs, blockAccepted>>

\* RMFaultySendCV1 describes sending CV1 message by the faulty node r. To reduce
\* the number of reachable states, the target view of this message is restricted by
\* the next one.
RMFaultySendCV1(r) ==
  /\ rmState[r].type = "bad"
  /\ LET cv == [type |-> "ChangeView1", rm |-> r, view |-> rmState[r].view, targetView |-> GetNewView(rmState[r].view), sourceView |-> rmState[r].view]
     IN /\ cv \notin msgs
        /\ msgs' = msgs \cup {cv}
        /\ UNCHANGED <<rmState, blockAccepted>>

\* RMFaultySendCV2 describes sending CV2 message by the faulty node r.  To reduce
\* the number of reachable states, the target view of this message is restricted by
\* the next one; the source view of this message is restricted by the current one.
RMFaultySendCV2(r) ==
  /\ rmState[r].type = "bad"
  /\ LET cv == [type |-> "ChangeView2", rm |-> r, view |-> rmState[r].view, targetView |-> GetNewView(rmState[r].view), sourceView |-> rmState[r].view]
     IN /\ cv \notin msgs
        /\ msgs' = msgs \cup {cv}
        /\ UNCHANGED <<rmState, blockAccepted>>

\* RMFaultyDoCV describes view changing by the faulty node r.
RMFaultyDoCV(r) ==
  /\ rmState[r].type = "bad"
  /\ rmState' = [rmState EXCEPT ![r].view = GetNewView(rmState[r].view)]
  /\ UNCHANGED <<msgs, blockAccepted>>

\* RMFaultySendPReq describes sending PrepareRequest message by the primary faulty node r.
\* To reduce the number of reachable states, the sourceView is always restricted by the
\* current r's view.
RMFaultySendPReq(r) ==
  /\ rmState[r].type = "bad"
  /\ IsPrimary(r)
  /\ LET pReq == [type |-> "PrepareRequest", rm |-> r, view |-> rmState[r].view, targetView |-> 0, sourceView |-> rmState[r].view]
     IN /\ pReq \notin msgs
        /\ msgs' = msgs \cup {pReq}
        /\ UNCHANGED <<rmState, blockAccepted>>

\* RMFaultySendPResp describes sending PrepareResponse message by the non-primary faulty node r.
\* To reduce the number of reachable states, the sourceView is always restricted by the
\* current r's view.
RMFaultySendPResp(r) ==
  /\ rmState[r].type = "bad"
  /\ \neg IsPrimary(r)
  /\ LET pResp == [type |-> "PrepareResponse", rm |-> r, view |-> rmState[r].view, targetView |-> 0, sourceView |-> rmState[r].view]
     IN /\ pResp \notin msgs
        /\ msgs' = msgs \cup {pResp}
        /\ UNCHANGED <<rmState, blockAccepted>>

\* RMFaultySendCommit describes sending Commit message by the faulty node r.
\* To reduce the number of reachable states, the sourceView is always restricted by the
\* current r's view.
RMFaultySendCommit(r) ==
  /\ rmState[r].type = "bad"
  /\ LET commit == [type |-> "Commit", rm |-> r, view |-> rmState[r].view, targetView |-> 0, sourceView |-> rmState[r].view]
     IN /\ commit \notin msgs
        /\ msgs' = msgs \cup {commit}
        /\ UNCHANGED <<rmState, blockAccepted>>

\* We don't describe sending DoCV messages by faulty node, because it can't
\* actually produce other than valid message, and valid message sending is described
\* in the "good" node specification. We also don't describe receiving the DoCV message
\* by the faulty node because it has a separate RMFaultyDoCV action enabled.

\* RMDie describes node r that was removed from the network at the particular step
\* of the behaviour. After this node r can't change its state and accept/send messages.
RMDie(r) ==
  /\ r \in RMDead
  /\ Cardinality({rm \in RM : rmState[rm].type = "dead"}) < F
  /\ rmState' = [rmState EXCEPT ![r].type = "dead"]
  /\ UNCHANGED <<msgs, blockAccepted>>

\* Terminating is an action that allows infinite stuttering to prevent deadlock on
\* behaviour termination. We consider termination to be valid if at least M nodes
\* have the block being accepted.
Terminating ==
  /\ Cardinality({rm \in RM : rmState[rm].type = "blockAccepted"}) >= M
  /\ UNCHANGED vars
  

\* Next is the next-state action describing the transition from the current state
\* to the next state of the behaviour.
Next ==
  \/ Terminating
  \/ \E r \in RM: 
       RMSendPrepareRequest(r) \/ RMSendPrepareResponse(r) \/ RMSendCommit(r)
         \/ RMAcceptBlock(r)
         \/ RMSendChangeView1(r) \/ RMSendChangeView1FromCV1(r)
         \/ RMSendChangeView2(r) \/ RMSendChangeView2FromCV2(r)
         \/ RMSendDoCV1ByLeader(r) \/ RMReceiveDoCV1FromLeader(r)
         \/ RMSendDoCV2ByLeader(r) \/ RMReceiveDoCV2FromLeader(r)
         \/ RMBeBad(r)
         \/ RMFaultySendCV1(r) \/ RMFaultySendCV2(r) \/ RMFaultyDoCV(r) \/ RMFaultySendCommit(r) \/ RMFaultySendPReq(r) \/ RMFaultySendPResp(r)
         \/ RMDie(r) \/ RMFetchBlock(r)

\* Safety is a temporal formula that describes the whole set of allowed
\* behaviours. It specifies only what the system MAY do (i.e. the set of
\* possible allowed behaviours for the system). It asserts only what may
\* happen; any behaviour that violates it does so at some point and
\* nothing past that point makes difference.
\*
\* E.g. this safety formula (applied standalone) allows the behaviour to end
\* with an infinite set of stuttering steps (those steps that DO NOT change
\* neither msgs nor rmState) and never reach the state where at least one
\* node is committed or accepted the block.
\*
\* To forbid such behaviours we must specify what the system MUST
\* do. It will be specified below with the help of fairness conditions in
\* the Fairness formula.
Safety == Init /\ [][Next]_vars


\* -------------- Fairness temporal formula --------------

\* Fairness is a temporal assumptions under which the model is working.
\* Usually it specifies different kind of assumptions for each/some
\* subactions of the Next's state action, but the only think that bothers
\* us is preventing infinite stuttering at those steps where some of Next's
\* subactions are enabled. Thus, the only thing that we require from the
\* system is to keep take the steps until it's impossible to take them.
\* That's exactly how the weak fairness condition works: if some action
\* remains continuously enabled, it must eventually happen.
Fairness == WF_vars(Next)

\* -------------- Specification --------------

\* The complete specification of the protocol written as a temporal formula.
Spec == Safety /\ Fairness

\* -------------- Liveness temporal formula --------------

\* For every possible behaviour it's true that eventually (i.e. at least once
\* through the behaviour) block will be accepted. It is something that dBFT
\* must guarantee (an in practice this condition is violated).
TerminationRequirement == <>(Cardinality({r \in RM : rmState[r].type = "blockAccepted"}) >= M)

\* A liveness temporal formula asserts only what must happen (i.e. specifies
\* what the system MUST do). Any behaviour can NOT violate it at ANY point;
\* there's always the rest of the behaviour that can always make the liveness
\* formula true; if there's no such behaviour than the liveness formula is
\* violated. The liveness formula is supposed to be checked as a property
\* by the TLC model checker.
Liveness == TerminationRequirement

\* -------------- ModelConstraints --------------

\* MaxViewConstraint is a state predicate restricting the number of possible
\* behaviour states. It is needed to reduce model checking time and prevent
\* the model graph size explosion. This formulae must be specified at the
\* "State constraint" section of the "Additional Spec Options" section inside
\* the model overview.
MaxViewConstraint == /\ \A r \in RM : rmState[r].view <= MaxView
                     /\ \A msg \in msgs : msg.targetView = 0 \/ msg.targetView <= MaxView + 1
                     
\* -------------- Invariants of the specification --------------

\* Model invariant is a state predicate (statement) that must be true for
\* every step of every reachable behaviour. Model invariant is supposed to
\* be checked as an Invariant by the TLC Model Checker.

\* TypeOK is a type-correctness invariant. It states that all elements of
\* specification variables must have the proper type throughout the behaviour.
TypeOK ==
  /\ rmState \in [RM -> RMStates]
  /\ msgs \subseteq Messages
  /\ blockAccepted \in [RM -> Nat]

\* InvTwoBlocksAcceptedAdvanced ensures that the proposed and accepted block
\* originally comes from the same view for every node that has the block
\* being accepted.
InvTwoBlocksAcceptedAdvanced == \A r1 \in RM:
                                \A r2 \in RM \ {r1}:
                                \/ rmState[r1].type /= "blockAccepted"
                                \/ rmState[r2].type /= "blockAccepted"
                                \/ blockAccepted[r1] = blockAccepted[r2]

\* InvFaultNodesCount states that there can be F faulty or dead nodes at max.
InvFaultNodesCount == Cardinality({
                                    r \in RM : rmState[r].type = "bad" \/ rmState[r].type = "dead"
                                 }) <= F

\* This theorem asserts the truth of the temporal formula whose meaning is that
\* the state predicates TypeOK, InvTwoBlocksAccepted and InvFaultNodesCount are
\* the invariants of the specification Spec. This theorem is not supposed to be
\* checked by the TLC model checker, it's here for the reader's understanding of
\* the purpose of TypeOK, InvTwoBlocksAccepted and InvFaultNodesCount.
THEOREM Spec => [](TypeOK /\ InvTwoBlocksAcceptedAdvanced /\ InvFaultNodesCount)
=============================================================================
\* Modification History
\* Last modified Fri Mar 03 10:51:05 MSK 2023 by root
\* Last modified Wed Feb 15 15:43:25 MSK 2023 by anna
\* Last modified Mon Jan 23 21:49:06 MSK 2023 by rik
\* Created Thu Dec 15 16:06:17 MSK 2022 by anna
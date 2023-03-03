-------------------------------- MODULE dbftCV3 --------------------------------

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
  msgs

\* vars is a tuple of all variables used in the specification. It is needed to
\* simplify fairness conditions definition.
vars == <<rmState, msgs>>

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
              type: {"initialized", "prepareSent", "commitSent", "blockAccepted", "cv1", "cv2", "cv3", "bad", "dead"},
              view : Nat
            ]

\* Messages is a set of records where each record holds the message type,
\* the message sender and sender's view by the moment when message was sent.
Messages == [type : {"PrepareRequest", "PrepareResponse", "Commit", "ChangeView1", "ChangeView2", "ChangeView3"}, rm : RM, view : Nat]

\* -------------- Useful operators --------------

\* IsPrimary is an operator defining whether provided node r is primary
\* for the current round from the r's point of view. It is a mapping
\* from RM to the set of {TRUE, FALSE}.
IsPrimary(r) == rmState[r].view % N = r

\* GetPrimary is an operator defining mapping from round index to the RM that
\* is primary in this round.
GetPrimary(view) == CHOOSE r \in RM : view % N = r

\* GetNewView returns new view number based on the previous node view value.
\* Current specifications only allows to increment view.
GetNewView(oldView) == oldView + 1

\* PrepareRequestSentOrReceived denotes whether there's a PrepareRequest
\* message received from the current round's speaker (as the node r sees it).
PrepareRequestSentOrReceived(r) == [type |-> "PrepareRequest", rm |-> GetPrimary(rmState[r].view), view |-> rmState[r].view] \in msgs

\* -------------- Safety temporal formula --------------

\* Init is the initial predicate initializing values at the start of every
\* behaviour.
Init ==
  /\ rmState = [r \in RM |-> [type |-> "initialized", view |-> 0]]
  /\ msgs = {}

\* RMSendPrepareRequest describes the primary node r broadcasting PrepareRequest.
RMSendPrepareRequest(r) ==
  /\ rmState[r].type = "initialized"
  /\ IsPrimary(r)
  /\ rmState' = [rmState EXCEPT ![r].type = "prepareSent"]
  /\ msgs' = msgs \cup {[type |-> "PrepareRequest", rm |-> r, view |-> rmState[r].view]}
  /\ UNCHANGED <<>>

\* RMSendPrepareResponse describes non-primary node r receiving PrepareRequest from
\* the primary node of the current round (view) and broadcasting PrepareResponse.
\* This step assumes that PrepareRequest always contains valid transactions and
\* signatures.
RMSendPrepareResponse(r) ==
  /\ rmState[r].type = "initialized"
  /\ \neg IsPrimary(r)
  /\ PrepareRequestSentOrReceived(r)
  /\ rmState' = [rmState EXCEPT ![r].type = "prepareSent"]
  /\ msgs' = msgs \cup {[type |-> "PrepareResponse", rm |-> r, view |-> rmState[r].view]}
  /\ UNCHANGED <<>>

\* RMSendCommit describes node r sending Commit if there's enough PrepareRequest/PrepareResponse
\* messages and no node has sent the ChangeView3 message. It is possible to send the Commit after
\* the ChangeView1 or ChangeView2 message was sent with additional constraints.
RMSendCommit(r) ==
  /\ \/ rmState[r].type = "prepareSent"
     \/ rmState[r].type = "cv1"
     \/ /\ rmState[r].type = "cv2"
        /\ Cardinality({
                         msg \in msgs : msg.type = "Commit" /\ msg.view = rmState[r].view
                      }) > F
  /\ Cardinality({
                   msg \in msgs : (msg.type = "PrepareResponse" \/ msg.type = "PrepareRequest") /\ msg.view = rmState[r].view
                }) >= M
  /\ Cardinality({
                   msg \in msgs : msg.type = "ChangeView3" /\ msg.view = rmState[r].view
                }) = 0
  /\ PrepareRequestSentOrReceived(r)
  /\ rmState' = [rmState EXCEPT ![r].type = "commitSent"]
  /\ msgs' = msgs \cup {[type |-> "Commit", rm |-> r, view |-> rmState[r].view]}
  /\ UNCHANGED <<>>

\* RMAcceptBlock describes node r collecting enough Commit messages and accepting
\* the block.
RMAcceptBlock(r) ==
  /\ rmState[r].type /= "bad"
  /\ rmState[r].type /= "dead"
  /\ rmState[r].type /= "blockAccepted"
  /\ PrepareRequestSentOrReceived(r)
  /\ Cardinality({
                   msg \in msgs : msg.type = "Commit" /\ msg.view = rmState[r].view
                }) >= M
  /\ rmState' = [rmState EXCEPT ![r].type = "blockAccepted"]
  /\ UNCHANGED <<msgs>>

\* FetchBlock describes node r that fetches the accepted block from some other node.
RMFetchBlock(r) ==
  /\ rmState[r].type /= "bad"
  /\ rmState[r].type /= "dead"
  /\ rmState[r].type /= "blockAccepted"
  /\ \E rmAccepted \in RM : /\ rmState[rmAccepted].type = "blockAccepted"
                            /\ rmState' = [rmState EXCEPT ![r].type = "blockAccepted", ![r].view = rmState[rmAccepted].view]
                            /\ UNCHANGED <<msgs>>

\* RMSendChangeView1 describes node r sending ChangeView1 message on timeout.
\* Only non-primary node is allowed to send ChangeView1 message, as the primary
\* must send the PrepareRequest if the timer fires.
RMSendChangeView1(r) ==
  /\ rmState[r].type = "initialized"
  /\ \neg IsPrimary(r)
  /\ rmState' = [rmState EXCEPT ![r].type = "cv1"]
  /\ msgs' = msgs \cup {[type |-> "ChangeView1", rm |-> r, view |-> rmState[r].view]}

\* RMSendChangeView2 describes node r sending ChangeView2 message on timeout either from
\* "cv1" state or after the node has sent the PrepareRequest or PrepareResponse message.
RMSendChangeView2(r) ==
  /\ \/ /\ rmState[r].type = "prepareSent"
        /\ Cardinality({
                        msg \in msgs : msg.type = "ChangeView1" /\ msg.view = rmState[r].view
                      }) > 0
     \/ rmState[r].type = "cv1"
  /\ Cardinality({
                   msg \in msgs : (msg.type = "ChangeView1" \/ msg.type = "PrepareRequest" \/ msg.type = "PrepareResponse") /\ msg.view = rmState[r].view
                }) >= M
  /\ \/ Cardinality({
                      msg \in msgs : msg.type = "Commit" /\ msg.view = rmState[r].view
                   }) <= F
     \/ Cardinality({
                     msg \in msgs : msg.type = "ChangeView3" /\ msg.view = rmState[r].view
                   }) > 0
  /\ rmState' = [rmState EXCEPT ![r].type = "cv2"]
  /\ msgs' = msgs \cup {[type |-> "ChangeView2", rm |-> r, view |-> rmState[r].view]}

\* RMSendChangeView3 describes node r sending ChangeView3 message on timeout either from
\* "cv2" state or after the node has sent the Commit message.
RMSendChangeView3(r) ==
  /\ \/ rmState[r].type = "cv2"
     \/ rmState[r].type = "commitSent"
  /\ Cardinality({msg \in msgs : (msg.type = "ChangeView2" \/ msg.type = "Commit") /\ msg.view = rmState[r].view}) >= M
  /\ Cardinality({msg \in msgs : (msg.type = "ChangeView2") /\ msg.view = rmState[r].view}) > 0
  /\ Cardinality({msg \in msgs : msg.type = "Commit" /\ msg.view = rmState[r].view}) <= F
  /\ rmState' = [rmState EXCEPT ![r].type = "cv3"]
  /\ msgs' = msgs \cup {[type |-> "ChangeView3", rm |-> r, view |-> rmState[r].view]}

\* RMReceiveChangeView describes node r receiving enough ChangeView[1,2,3] messages for
\* view changing.
RMReceiveChangeView(r) ==
  /\ rmState[r].type /= "bad"
  /\ rmState[r].type /= "dead"
  /\ rmState[r].type /= "blockAccepted"
  /\ \/ Cardinality({rm \in RM : Cardinality({msg \in msgs : /\ msg.rm = rm
                                                             /\ msg.type = "ChangeView1"
                                                             /\ GetNewView(msg.view) >= GetNewView(rmState[r].view)
                                            }) # 0
                   }) >= M
     \/ Cardinality({rm \in RM : Cardinality({msg \in msgs : /\ msg.rm = rm
                                                             /\ msg.type = "ChangeView2"
                                                             /\ GetNewView(msg.view) >= GetNewView(rmState[r].view)
                                            }) # 0
                   }) >= M
     \/ Cardinality({rm \in RM : Cardinality({msg \in msgs : /\ msg.rm = rm
                                                             /\ msg.type = "ChangeView3"
                                                             /\ GetNewView(msg.view) >= GetNewView(rmState[r].view)
                                            }) # 0
                   }) >= M
  /\ rmState' = [rmState EXCEPT ![r].type = "initialized", ![r].view = GetNewView(rmState[r].view)]
  /\ UNCHANGED <<msgs>>

\* RMBeBad describes the faulty node r that will send any kind of consensus message starting
\* from the step it's gone wild. This step is enabled only when RMFault is non-empty set.
RMBeBad(r) ==
  /\ r \in RMFault
  /\ Cardinality({rm \in RM : rmState[rm].type = "bad"}) < F
  /\ rmState' = [rmState EXCEPT ![r].type = "bad"]
  /\ UNCHANGED <<msgs>>

\* RMFaultySendCV describes sending CV1 message by the faulty node r.
RMFaultySendCV1(r) ==
  /\ rmState[r].type = "bad"
  /\ LET cv == [type |-> "ChangeView1", rm |-> r, view |-> rmState[r].view]
     IN /\ cv \notin msgs
        /\ msgs' = msgs \cup {cv}
        /\ UNCHANGED <<rmState>>

\* RMFaultySendCV2 describes sending CV2 message by the faulty node r.
RMFaultySendCV2(r) ==
  /\ rmState[r].type = "bad"
  /\ LET cv == [type |-> "ChangeView2", rm |-> r, view |-> rmState[r].view]
     IN /\ cv \notin msgs
        /\ msgs' = msgs \cup {cv}
        /\ UNCHANGED <<rmState>>

\* RMFaultySendCV3 describes sending CV3 message by the faulty node r.
RMFaultySendCV3(r) ==
  /\ rmState[r].type = "bad"
  /\ LET cv == [type |-> "ChangeView3", rm |-> r, view |-> rmState[r].view]
     IN /\ cv \notin msgs
        /\ msgs' = msgs \cup {cv}
        /\ UNCHANGED <<rmState>>

\* RMFaultyDoCV describes view changing by the faulty node r.
RMFaultyDoCV(r) ==
  /\ rmState[r].type = "bad"
  /\ rmState' = [rmState EXCEPT ![r].view = GetNewView(rmState[r].view)]
  /\ UNCHANGED <<msgs>>

\* RMFaultySendPReq describes sending PrepareRequest message by the primary faulty node r.
RMFaultySendPReq(r) ==
  /\ rmState[r].type = "bad"
  /\ IsPrimary(r)
  /\ LET pReq == [type |-> "PrepareRequest", rm |-> r, view |-> rmState[r].view]
     IN /\ pReq \notin msgs
        /\ msgs' = msgs \cup {pReq}
        /\ UNCHANGED <<rmState>>

\* RMFaultySendPResp describes sending PrepareResponse message by the non-primary faulty node r.
RMFaultySendPResp(r) ==
  /\ rmState[r].type = "bad"
  /\ \neg IsPrimary(r)
  /\ LET pResp == [type |-> "PrepareResponse", rm |-> r, view |-> rmState[r].view]
     IN /\ pResp \notin msgs
        /\ msgs' = msgs \cup {pResp}
        /\ UNCHANGED <<rmState>>

\* RMFaultySendCommit describes sending Commit message by the faulty node r.
RMFaultySendCommit(r) ==
  /\ rmState[r].type = "bad"
  /\ LET commit == [type |-> "Commit", rm |-> r, view |-> rmState[r].view]
     IN /\ commit \notin msgs
        /\ msgs' = msgs \cup {commit}
        /\ UNCHANGED <<rmState>>

\* RMDie describes node r that was removed from the network at the particular step
\* of the behaviour. After this node r can't change its state and accept/send messages.
RMDie(r) ==
  /\ r \in RMDead
  /\ Cardinality({rm \in RM : rmState[rm].type = "dead"}) < F
  /\ rmState' = [rmState EXCEPT ![r].type = "dead"]
  /\ UNCHANGED <<msgs>>

\* Terminating is an action that allows infinite stuttering to prevent deadlock on
\* behaviour termination. We consider termination to be valid if at least M nodes
\* has the block being accepted.
Terminating ==
  /\ Cardinality({rm \in RM : rmState[rm].type = "blockAccepted"}) >= M
  /\ UNCHANGED <<msgs, rmState>>

\* The next-state action.
Next ==
  \/ Terminating
  \/ \E r \in RM:
       RMSendPrepareRequest(r) \/ RMSendPrepareResponse(r) \/ RMSendCommit(r)
         \/ RMAcceptBlock(r) \/ RMSendChangeView1(r) \/ RMReceiveChangeView(r) \/ RMBeBad(r) \/ RMSendChangeView2(r) \/ RMSendChangeView3(r)
         \/ RMFaultySendCV1(r) \/ RMFaultyDoCV(r) \/ RMFaultySendCommit(r) \/ RMFaultySendPReq(r) \/ RMFaultySendPResp(r) \/ RMFaultySendCV2(r) \/ RMFaultySendCV3(r)
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
                     /\ \A msg \in msgs : msg.view <= MaxView

\* -------------- Invariants of the specification --------------

\* Model invariant is a state predicate (statement) that must be true for
\* every step of every reachable behaviour. Model invariant is supposed to
\* be checked as an Invariant by the TLC Model Checker.

\* TypeOK is a type-correctness invariant. It states that all elements of
\* specification variables must have the proper type throughout the behaviour.
TypeOK ==
  /\ rmState \in [RM -> RMStates]
  /\ msgs \subseteq Messages

\* InvTwoBlocksAccepted states that there can't be two different blocks accepted in
\* the two different views, i.e. dBFT must not allow forks.
InvTwoBlocksAccepted == \A r1 \in RM:
                  \A r2 \in RM \ {r1}:
                  \/ rmState[r1].type /= "blockAccepted"
                  \/ rmState[r2].type /= "blockAccepted"
                  \/ rmState[r1].view = rmState[r2].view

\* InvFaultNodesCount states that there can be F faulty or dead nodes at max.
InvFaultNodesCount == Cardinality({
                                    r \in RM : rmState[r].type = "bad" \/ rmState[r].type = "dead"
                                 }) <= F

\* This theorem asserts the truth of the temporal formula whose meaning is that
\* the state predicates TypeOK, InvTwoBlocksAccepted and InvFaultNodesCount are
\* the invariants of the specification Spec. This theorem is not supposed to be
\* checked by the TLC model checker, it's here for the reader's understanding of
\* the purpose of TypeOK, InvTwoBlocksAccepted and InvFaultNodesCount.
THEOREM Spec => [](TypeOK /\ InvTwoBlocksAccepted /\ InvFaultNodesCount)

=============================================================================
\* Modification History
\* Last modified Wed Mar 01 12:11:07 MSK 2023 by root
\* Last modified Tue Feb 07 23:11:19 MSK 2023 by rik
\* Last modified Fri Feb 03 18:09:33 MSK 2023 by anna
\* Created Thu Dec 15 16:06:17 MSK 2022 by anna

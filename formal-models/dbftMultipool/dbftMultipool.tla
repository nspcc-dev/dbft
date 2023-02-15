---------------------------- MODULE dbftMultipool ----------------------------

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
  MaxView,

  \* MaxUndeliveredMessages is the maximum number of messages in the common
  \* messages pool (msgs) that were not received and handled by all consensus
  \* nodes. It must not be too small (>= 3) in order to allow model taking
  \* at least any steps. At the same time it must not be too high (<= 6 is
  \* recommended) in order to avoid states graph size explosion.
  MaxUndeliveredMessages

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
  /\ MaxUndeliveredMessages \in Nat
  /\ MaxUndeliveredMessages >= 3 \* First value when block can be accepted in some behaviours.

\* Messages is a set of records where each record holds the message type,
\* the message sender and sender's view by the moment when message was sent.
Messages == [type : {"PrepareRequest", "PrepareResponse", "Commit", "ChangeView"}, rm : RM, view : Nat]

\* RMStates is a set of records where each record holds the node state, the node current view
\* and the pool of messages the nde has been sent or received and handled.
RMStates == [
              type: {"initialized", "prepareSent", "commitSent", "blockAccepted", "bad", "dead"},
              view : Nat,
              pool : SUBSET Messages
            ]

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

\* IsViewChanging denotes whether node r have sent ChangeView message for the
\* current (or later) round.
IsViewChanging(r) == Cardinality({msg \in rmState[r].pool : msg.type = "ChangeView" /\ msg.view >= rmState[r].view /\ msg.rm = r}) /= 0

\* CountCommitted returns the number of nodes that have sent the Commit message
\* in the current round (as the node r sees it).
CountCommitted(r) == Cardinality({rm \in RM : Cardinality({msg \in rmState[r].pool : msg.rm = rm /\ msg.type = "Commit"}) /= 0})

\* CountFailed returns the number of nodes that haven't sent any message since
\* the last round (as the node r sees it from the point of its pool).
CountFailed(r) == Cardinality({rm \in RM : Cardinality({msg \in rmState[r].pool : msg.rm = rm /\ msg.view >= rmState[r].view}) = 0 })

\* MoreThanFNodesCommittedOrLost denotes whether more than F nodes committed or
\* failed to communicate in the current round.
MoreThanFNodesCommittedOrLost(r) == CountCommitted(r) + CountFailed(r) > F

\* NotAcceptingPayloadsDueToViewChanging returns whether the node doesn't accept
\* payloads in the current step.
NotAcceptingPayloadsDueToViewChanging(r) ==
  /\ IsViewChanging(r)
  /\ \neg MoreThanFNodesCommittedOrLost(r)

\* PrepareRequestSentOrReceived denotes whether there's a PrepareRequest
\* message received from the current round's speaker (as the node r sees it).
PrepareRequestSentOrReceived(r) == [type |-> "PrepareRequest", rm |-> GetPrimary(rmState[r].view), view |-> rmState[r].view] \in rmState[r].pool

\* CommitSent returns whether the node has its commit sent for the current block.
CommitSent(r) == Cardinality({msg \in rmState[r].pool : msg.rm = r /\ msg.type = "Commit"}) > 0

\* -------------- Safety temporal formula --------------

\* Init is the initial predicate initializing values at the start of every
\* behaviour.
Init ==
  /\ rmState = [r \in RM |-> [type |-> "initialized", view |-> 1, pool |-> {}]]
  /\ msgs = {}

\* RMSendPrepareRequest describes the primary node r broadcasting PrepareRequest.
RMSendPrepareRequest(r) ==
  /\ rmState[r].type = "initialized"
  /\ IsPrimary(r)
  /\ LET pReq == [type |-> "PrepareRequest", rm |-> r, view |-> rmState[r].view]
         commit == [type |-> "Commit", rm |-> r, view |-> rmState[r].view]
     IN /\ pReq \notin msgs
        /\ IF Cardinality({m \in rmState[r].pool : m.type = "PrepareResponse" /\ m.view = rmState[r].view}) < M - 1 \* -1 is for the current PrepareRequest.
           THEN /\ rmState' = [rmState EXCEPT ![r].type = "prepareSent", ![r].pool = rmState[r].pool \cup {pReq}]
                /\ msgs' = msgs \cup {pReq}
           ELSE /\ msgs' = msgs \cup {pReq, commit}
                /\ IF Cardinality({m \in rmState[r].pool : m.type = "Commit" /\ m.view = rmState[r].view}) < M-1 \* -1 is for the current Commit
                   THEN rmState' = [rmState EXCEPT ![r].type = "commitSent", ![r].pool = rmState[r].pool \cup {pReq, commit}]
                   ELSE rmState' = [rmState EXCEPT ![r].type = "blockAccepted", ![r].pool = rmState[r].pool \cup {pReq, commit}]
        /\ UNCHANGED <<>>

\* RMSendChangeView describes node r sending ChangeView message on timeout.
RMSendChangeView(r) ==
  /\ rmState[r].type /= "bad"
  /\ rmState[r].type /= "dead"
  /\ rmState[r].type /= "blockAccepted"
  /\ \/ (IsPrimary(r) /\ PrepareRequestSentOrReceived(r))
     \/ (\neg IsPrimary(r) /\ \neg CommitSent(r))
  /\ LET msg == [type |-> "ChangeView", rm |-> r, view |-> rmState[r].view]
     IN /\ msg \notin msgs
        /\ msgs' = msgs \cup {msg}
        /\ IF Cardinality({m \in rmState[r].pool : m.type = "ChangeView" /\ GetNewView(m.view) >= GetNewView(msg.view)}) >= M-1 \* -1 is for the currently sent CV
           THEN rmState' = [rmState EXCEPT ![r].type = "initialized", ![r].view = GetNewView(msg.view), ![r].pool = rmState[r].pool \cup {msg}]
           ELSE rmState' = [rmState EXCEPT ![r].pool = rmState[r].pool \cup {msg}]

\* OnTimeout describes two actions the node can take on timeout for waiting any event.
OnTimeout(r) ==
  \/ RMSendPrepareRequest(r)
  \/ RMSendChangeView(r)

\* RMOnPrepareRequest describes non-primary node r receiving PrepareRequest from the
\* primary node of the current round (view) and broadcasts PrepareResponse.
\* This step assumes that PrepareRequest always contains valid transactions and
\* signatures.
RMOnPrepareRequest(r) ==
  /\ rmState[r].type = "initialized"
  /\ \E msg \in msgs \ rmState[r].pool:
        /\ msg.rm /= r
        /\ msg.type = "PrepareRequest"
        /\ msg.view = rmState[r].view
        /\ \neg IsPrimary(r)
        /\ \neg NotAcceptingPayloadsDueToViewChanging(r) \* dbft.go -L296, in C# node, but not in ours
        /\ LET pResp == [type |-> "PrepareResponse", rm |-> r, view |-> rmState[r].view]
               commit == [type |-> "Commit", rm |-> r, view |-> rmState[r].view]
           IN IF Cardinality({m \in rmState[r].pool : m.type = "PrepareResponse" /\ m.view = rmState[r].view}) < M - 1 - 1 \* -1 is for reveived PrepareRequest; -1 is for current PrepareResponse
              THEN /\ rmState' = [rmState EXCEPT ![r].type = "prepareSent", ![r].pool = rmState[r].pool \cup {msg, pResp}]
                   /\ msgs' = msgs \cup {pResp}
              ELSE /\ msgs' = msgs \cup {msg, pResp, commit}
                   /\ IF Cardinality({m \in rmState[r].pool : m.type = "Commit" /\ m.view = rmState[r].view}) < M-1 \* -1 is for the current Commit
                      THEN rmState' = [rmState EXCEPT ![r].type = "commitSent", ![r].pool = rmState[r].pool \cup {msg, pResp, commit}]
                      ELSE rmState' = [rmState EXCEPT ![r].type = "blockAccepted", ![r].pool = rmState[r].pool \cup {msg, pResp, commit}]
        /\ UNCHANGED <<>>

\* RMOnPrepareResponse describes node r accepting PrepareResponse message and handling it.
\* If there's enough PrepareResponses collected it will send the Commit; in case if there's
\* enough Commits it will accept the block.
RMOnPrepareResponse(r) ==
  /\ rmState[r].type /= "bad"
  /\ rmState[r].type /= "dead"
  /\ rmState[r].type /= "blockAccepted"
  /\ \E msg \in msgs \ rmState[r].pool:
        /\ msg.rm /= r
        /\ msg.type = "PrepareResponse"
        /\ msg.view = rmState[r].view
        /\ \neg NotAcceptingPayloadsDueToViewChanging(r)
        /\ IF \/ Cardinality({m \in rmState[r].pool : (m.type = "PrepareRequest" \/ m.type = "PrepareResponse") /\ m.view = rmState[r].view}) < M - 1 \* -1 is for the currently received PrepareResponse.
              \/ CommitSent(r)
              \/ \neg PrepareRequestSentOrReceived(r)
           THEN /\ rmState' = [rmState EXCEPT ![r].pool = rmState[r].pool \cup {msg}]
                /\ UNCHANGED <<msgs>>
           ELSE LET commit == [type |-> "Commit", rm |-> r, view |-> rmState[r].view]
                IN /\ msgs' = msgs \cup {msg, commit}
                   /\ IF Cardinality({m \in rmState[r].pool : m.type = "Commit" /\ m.view = rmState[r].view}) < M-1 \* -1 is for the current Commit
                      THEN rmState' = [rmState EXCEPT ![r].type = "commitSent", ![r].pool = rmState[r].pool \cup {msg, commit}]
                      ELSE rmState' = [rmState EXCEPT ![r].type = "blockAccepted", ![r].pool = rmState[r].pool \cup {msg, commit}]

\* RMOnCommit describes node r accepting Commit message and (in case if there's enough Commits)
\* accepting the block.
RMOnCommit(r) ==
  /\ rmState[r].type /= "bad"
  /\ rmState[r].type /= "dead"
  /\ rmState[r].type /= "blockAccepted"
  /\ \E msg \in msgs \ rmState[r].pool:
        /\ msg.rm /= r
        /\ msg.type = "Commit"
        /\ msg.view = rmState[r].view
        /\ IF Cardinality({m \in rmState[r].pool : m.type = "Commit" /\ m.view = rmState[r].view}) < M-1 \* -1 is for the currently accepting commit
           THEN rmState' = [rmState EXCEPT ![r].pool = rmState[r].pool \cup {msg}]
           ELSE rmState' = [rmState EXCEPT ![r].type = "blockAccepted", ![r].pool = rmState[r].pool \cup {msg}]
        /\ UNCHANGED <<msgs>>

\* RMOnChangeView describes node r receiving ChangeView message and (in case if enough ChangeViews
\* is collected) changing its view.
RMOnChangeView(r) ==
  /\ rmState[r].type /= "bad"
  /\ rmState[r].type /= "dead"
  /\ rmState[r].type /= "blockAccepted"
  /\ \E msg \in msgs \ rmState[r].pool:
        /\ msg.rm /= r
        /\ msg.type = "ChangeView"
        /\ msg.view = rmState[r].view
        /\ \neg CommitSent(r)
        /\ Cardinality({m \in rmState[r].pool : m.type = "ChangeView" /\ m.rm = msg.rm /\ m.view > msg.view}) = 0
        /\ IF Cardinality({m \in rmState[r].pool : m.type = "ChangeView" /\ GetNewView(m.view) >= GetNewView(msg.view)}) < M-1 \* -1 is for the currently accepting CV
           THEN rmState' = [rmState EXCEPT ![r].pool = rmState[r].pool \cup {msg}]
           ELSE rmState' = [rmState EXCEPT ![r].type = "initialized", ![r].view = GetNewView(msg.view), ![r].pool = rmState[r].pool \cup {msg}]
        /\ UNCHANGED <<msgs>>

\* RMBeBad describes the faulty node r that will send any kind of consensus message starting
\* from the step it's gone wild. This step is enabled only when RMFault is non-empty set.
RMBeBad(r) ==
  /\ r \in RMFault
  /\ Cardinality({rm \in RM : rmState[rm].type = "bad"}) < F
  /\ rmState' = [rmState EXCEPT ![r].type = "bad"]
  /\ UNCHANGED <<msgs>>

\* RMFaultySendCV describes sending CV message by the faulty node r.
RMFaultySendCV(r) ==
  /\ rmState[r].type = "bad"
  /\ LET cv == [type |-> "ChangeView", rm |-> r, view |-> rmState[r].view]
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
  /\ Cardinality({rm \in RM : rmState[rm].type = "blockAccepted"}) >=1
  /\ UNCHANGED <<msgs, rmState>>

\* Next is the next-state action describing the transition from the current state
\* to the next state of the behaviour.
Next ==
  \/ Terminating
  \/ \E r \in RM :
       \/ OnTimeout(r)
       \/ RMOnPrepareRequest(r) \/ RMOnPrepareResponse(r) \/ RMOnCommit(r) \/ RMOnChangeView(r)
       \/ RMDie(r) \/ RMBeBad(r)
       \/ RMFaultySendCV(r) \/ RMFaultyDoCV(r) \/ RMFaultySendCommit(r) \/ RMFaultySendPReq(r) \/ RMFaultySendPResp(r)

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

\* For every possible behaviour it's true that there's at least one PrepareRequest
\* message from the speaker, there's at lest one PrepareResponse message and at
\* least one Commit message.
PrepareRequestSentRequirement == <>(\E msg \in msgs : msg.type = "PrepareRequest")
PrepareResponseSentRequirement == <>(\E msg \in msgs : msg.type = "PrepareResponse")
CommitSentRequirement == <>(\E msg \in msgs : msg.type = "Commit")

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
Liveness == /\ PrepareRequestSentRequirement
            /\ PrepareResponseSentRequirement
            /\ CommitSentRequirement
            /\ TerminationRequirement

\* -------------- Model constraints --------------

\* Model constraints are a set of state predicates restricting the number of possible
\* behaviour states. It is needed to reduce model checking time and prevent
\* the model graph size explosion. These formulaes must be specified at the
\* "State constraint" section of the "Additional Spec Options" section inside
\* the model overview.

\* MaxViewConstraint is a state predicate restricting the maximum view of messages
\* and consensus nodes state.
MaxViewConstraint == /\ \A r \in RM : rmState[r].view <= MaxView
                     /\ \A msg \in msgs : msg.view <= MaxView

\* MaxUndeliveredMessageConstraint is a state predicate restricting the maximum
\* number of messages undelivered to any of the consensus nodes.
MaxUndeliveredMessageConstraint == Cardinality({msg \in msgs : \E rm \in RM : msg \notin rmState[rm].pool}) <= MaxUndeliveredMessages

\* ModelConstraint is overall model constraint rule.
ModelConstraint == MaxViewConstraint /\ MaxUndeliveredMessageConstraint

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

\* InvDeadlock is a deadlock invariant, it states that the following situation expected
\* never to happen: one node is committed in a single view, two others are committed in
\* a larger view, and the last one has its view changing.
InvDeadlock == \A r1 \in RM :
               \A r2 \in RM \ {r1} :
               \A r3 \in RM \ {r1, r2} :
               \A r4 \in RM \ {r1, r2, r3} :
               \/ rmState[r1].type /= "commitSent"
               \/ rmState[r2].type /= "commitSent"
               \/ rmState[r3].type /= "commitSent"
               \/ \neg IsViewChanging(r4)
               \/ rmState[r1].view >= rmState[r2].view
               \/ rmState[r2].view /= rmState[r3].view
               \/ rmState[r3].view /= rmState[r4].view

\* InvFaultNodesCount states that there can be F faulty or dead nodes at max.
InvFaultNodesCount == Cardinality({
                                    r \in RM : rmState[r].type = "bad" \/ rmState[r].type = "dead"
                                 }) <= F

\* This theorem asserts the truth of the temporal formula whose meaning is that
\* the state predicates TypeOK, InvTwoBlocksAccepted, InvDeadlock and InvFaultNodesCount are
\* the invariants of the specification Spec. This theorem is not supposed to be
\* checked by the TLC model checker, it's here for the reader's understanding of
\* the purpose of TypeOK, InvTwoBlocksAccepted, InvDeadlock and InvFaultNodesCount.
THEOREM Spec => [](TypeOK /\ InvTwoBlocksAccepted /\ InvDeadlock /\ InvFaultNodesCount)

=============================================================================
\* Modification History
\* Last modified Fri Feb 17 15:51:19 MSK 2023 by anna
\* Created Tue Jan 10 12:28:45 MSK 2023 by anna

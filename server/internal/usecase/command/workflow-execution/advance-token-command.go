package workflow_execution

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/bpmn"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/server/internal/usecase/command/queue"
	"deadalus-orch/shared/models"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func init() {
	gob.Register(AdvanceTokenCommand{})
}

type AdvanceTokenCommand struct {
	ExecutionID string
	TokenID     string
	OutputData  map[string]interface{}
	CF          string
	CFS         string
}

func (cmd *AdvanceTokenCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if cmd.ExecutionID == "" || cmd.TokenID == "" {
		commandResult.Error = "ExecutionID and TokenID are required"
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}

	// 1. Fetch Repositories
	execRepo, err := db.NewWorkflowExecutionRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	tokenRepo, err := db.NewExecutionTokenRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	defRepo, err := db.NewWorkflowDefinitionRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	jobRepo, err := db.NewWorkflowJobRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// 2. Fetch Entities
	execution, err := execRepo.GetWorkflowExecutionByID(cmd.ExecutionID, now)
	if err != nil || execution == nil {
		commandResult.Error = fmt.Sprintf("execution not found: %s", cmd.ExecutionID)
		return *commandResult
	}

	if execution.Status != models.WorkflowExecutionStatusRunning && execution.Status != models.WorkflowExecutionStatusPending {
		commandResult.Result = execution
		return *commandResult
	}

	token, err := tokenRepo.GetExecutionTokenByID(cmd.TokenID, now)
	if err != nil || token == nil {
		commandResult.Error = fmt.Sprintf("token not found: %s", cmd.TokenID)
		return *commandResult
	}

	if token.Status != models.ExecutionTokenStatusActive && token.Status != models.ExecutionTokenStatusWaiting {
		commandResult.Result = execution
		return *commandResult
	}

	wasWaitingToken := token.Status == models.ExecutionTokenStatusWaiting
	if token.Status == models.ExecutionTokenStatusWaiting {
		token.Status = models.ExecutionTokenStatusActive
		tokenRepo.UpdateExecutionToken(token, now)
	}

	def, err := defRepo.GetWorkflowDefinitionByID(execution.WorkflowDefinitionID, now)
	if err != nil || def == nil {
		commandResult.Error = fmt.Sprintf("workflow definition not found: %s", execution.WorkflowDefinitionID)
		return *commandResult
	}

	// 3. Parse BPMN Model
	bpmnModel, err := bpmn.ParseBPMN(def.Payload)
	if err != nil {
		commandResult.Error = fmt.Sprintf("failed to parse BPMN: %s", err.Error())
		return *commandResult
	}

	// 4. Merge OutputData if provided
	if cmd.OutputData != nil {
		if execution.StateData == nil {
			execution.StateData = make(map[string]interface{})
		}
		for k, v := range cmd.OutputData {
			execution.StateData[k] = v
		}
	}

	dbResolver := &db.DBEnvResolver{
		UOW:        uow,
		TenantCF:   cmd.CF,
		TenantCFS:  cmd.CFS,
		VNamespace: execution.VNamespace,
		Now:        now,
	}

	// 5. Advance Token Loop
	maxSteps := 50
	for step := 0; step < maxSteps; step++ {
		currentNode := bpmnModel.GetNode(token.CurrentNodeID)
		if currentNode == nil {
			commandResult.Error = fmt.Sprintf("node not found in model: %s", token.CurrentNodeID)
			return *commandResult
		}

		switch currentNode.Type {
		case bpmn.ElementStartEvent:
			outgoing := bpmnModel.GetOutgoingFlows(currentNode.ID)
			if len(outgoing) == 0 {
				token.Status = models.ExecutionTokenStatusCompleted
				tokenRepo.UpdateExecutionToken(token, now)
				goto CheckExecutionCompletion
			}
			token.CurrentNodeID = outgoing[0].TargetRef
			tokenRepo.UpdateExecutionToken(token, now)

		case bpmn.ElementExclusiveGateway:
			outgoing := bpmnModel.GetOutgoingFlows(currentNode.ID)
			log.Info().
				Str("executionID", execution.ID).
				Str("tokenID", token.ID).
				Str("gatewayID", currentNode.ID).
				Str("gatewayName", currentNode.Name).
				Str("gatewayType", "ExclusiveGateway").
				Int("outgoingFlowsCount", len(outgoing)).
				Str("defaultFlowID", currentNode.DefaultFlowID).
				Msg("🔀 Evaluating Exclusive Gateway")

			var selectedFlow *bpmn.SequenceFlow
			var defaultFlowCandidate *bpmn.SequenceFlow

			// Validation: In a divergent Exclusive Gateway, any flow that is NOT the default flow MUST have a condition expression
			if len(outgoing) > 1 {
				for _, flow := range outgoing {
					isDefault := currentNode.DefaultFlowID != "" && flow.ID == currentNode.DefaultFlowID
					condStr := strings.TrimSpace(flow.Condition)
					if condStr == "" && !isDefault {
						errMsg := fmt.Sprintf("gateway configuration error for %s (%s): outgoing flow %s has no condition and is not marked as default flow", currentNode.ID, currentNode.Name, flow.ID)
						log.Error().
							Str("executionID", execution.ID).
							Str("gatewayID", currentNode.ID).
							Str("flowID", flow.ID).
							Msg("❌ Gateway error: outgoing flow without condition is not marked as default flow")

						token.Status = models.ExecutionTokenStatusCancelled
						tokenRepo.UpdateExecutionToken(token, now)

						execution.Status = models.WorkflowExecutionStatusFailed
						execution.Error = errMsg
						execution.CompletedAt = &now
						execRepo.UpdateWorkflowExecution(execution, now)

						commandResult.Error = errMsg
						commandResult.Result = execution
						return *commandResult
					}
				}
			}

			for idx, flow := range outgoing {
				if currentNode.DefaultFlowID != "" && flow.ID == currentNode.DefaultFlowID {
					defaultFlowCandidate = flow
					continue
				}

				condStr := strings.TrimSpace(flow.Condition)
				matched, err := bpmn.EvaluateConditionWithResolver(condStr, execution.StateData, dbResolver)

				if err != nil {
					log.Warn().
						Err(err).
						Str("executionID", execution.ID).
						Str("gatewayID", currentNode.ID).
						Str("flowID", flow.ID).
						Str("targetRef", flow.TargetRef).
						Str("condition", condStr).
						Msg("⚠️ Gateway condition evaluation returned error")
				} else {
					log.Info().
						Str("executionID", execution.ID).
						Str("gatewayID", currentNode.ID).
						Str("flowID", flow.ID).
						Str("targetRef", flow.TargetRef).
						Str("condition", condStr).
						Bool("evaluationPassed", matched).
						Int("flowIndex", idx).
						Msg("🔍 Evaluated flow condition")
				}

				if err == nil && matched {
					selectedFlow = flow
					break
				}
			}

			isDefaultTaken := false
			if selectedFlow == nil && defaultFlowCandidate != nil {
				selectedFlow = defaultFlowCandidate
				isDefaultTaken = true
				log.Info().
					Str("executionID", execution.ID).
					Str("gatewayID", currentNode.ID).
					Str("defaultFlowID", defaultFlowCandidate.ID).
					Str("targetRef", defaultFlowCandidate.TargetRef).
					Msg("↩️ No conditions satisfied. Taking designated default flow path")
			}

			if selectedFlow == nil {
				errMsg := fmt.Sprintf("gateway evaluation failed for %s (%s): no condition satisfied and no default flow defined", currentNode.ID, currentNode.Name)
				log.Error().
					Str("executionID", execution.ID).
					Str("gatewayID", currentNode.ID).
					Str("gatewayName", currentNode.Name).
					Msg("❌ Exclusive Gateway evaluation failed: no condition met and no default flow available")

				token.Status = models.ExecutionTokenStatusCancelled
				tokenRepo.UpdateExecutionToken(token, now)

				execution.Status = models.WorkflowExecutionStatusFailed
				execution.Error = errMsg
				execution.CompletedAt = &now
				execRepo.UpdateWorkflowExecution(execution, now)

				commandResult.Error = errMsg
				commandResult.Result = execution
				return *commandResult
			}

			log.Info().
				Str("executionID", execution.ID).
				Str("gatewayID", currentNode.ID).
				Str("targetNodeID", selectedFlow.TargetRef).
				Str("flowID", selectedFlow.ID).
				Bool("isDefaultTaken", isDefaultTaken).
				Msg("✅ Exclusive Gateway path selected successfully")

			token.CurrentNodeID = selectedFlow.TargetRef
			tokenRepo.UpdateExecutionToken(token, now)

		case bpmn.ElementParallelGateway:
			incoming := bpmnModel.GetIncomingFlows(currentNode.ID)
			outgoing := bpmnModel.GetOutgoingFlows(currentNode.ID)

			log.Info().
				Str("executionID", execution.ID).
				Str("tokenID", token.ID).
				Str("gatewayID", currentNode.ID).
				Str("gatewayName", currentNode.Name).
				Str("gatewayType", "ParallelGateway").
				Int("incomingCount", len(incoming)).
				Int("outgoingCount", len(outgoing)).
				Msg("🔀 Processing Parallel Gateway")

			if len(incoming) > 1 {
				// Parallel Join / Merge
				allTokens, _ := tokenRepo.GetTokensByExecutionID(execution.ID, now)
				arrivedCount := 0
				for _, t := range allTokens {
					if t.CurrentNodeID == currentNode.ID {
						arrivedCount++
					}
				}

				log.Info().
					Str("executionID", execution.ID).
					Str("gatewayID", currentNode.ID).
					Int("arrivedTokens", arrivedCount).
					Int("requiredTokens", len(incoming)).
					Msg("⚖️ Parallel Join Gateway status check")

				if arrivedCount < len(incoming) {
					token.Status = models.ExecutionTokenStatusWaiting
					tokenRepo.UpdateExecutionToken(token, now)

					log.Info().
						Str("executionID", execution.ID).
						Str("gatewayID", currentNode.ID).
						Msg("⏳ Parallel Join Gateway: Waiting for remaining branches to arrive")
					goto SaveExecutionState
				}

				token.Status = models.ExecutionTokenStatusCompleted
				tokenRepo.UpdateExecutionToken(token, now)

				// Complete all other branch tokens that were waiting at this join gateway
				for _, t := range allTokens {
					if t.CurrentNodeID == currentNode.ID && (t.Status == models.ExecutionTokenStatusWaiting || t.Status == models.ExecutionTokenStatusActive) {
						t.Status = models.ExecutionTokenStatusCompleted
						tokenRepo.UpdateExecutionToken(&t, now)
					}
				}

				log.Info().
					Str("executionID", execution.ID).
					Str("gatewayID", currentNode.ID).
					Msg("✅ All incoming parallel branches arrived! Merging into single token")

				if len(outgoing) > 0 {
					mergedTokenID := strings.ReplaceAll(uuid.New().String(), "-", "")
					mergedToken := &models.ExecutionToken{
						ID:                   mergedTokenID,
						WorkflowExecutionID: execution.ID,
						WorkflowDefinitionID: def.ID,
						VNamespace:           execution.VNamespace,
						CurrentNodeID:        outgoing[0].TargetRef,
						Status:               models.ExecutionTokenStatusActive,
						ParentTokenID:        token.ID,
						CreatedAt:            now,
						UpdatedAt:            now,
					}
					tokenRepo.CreateExecutionToken(mergedToken, now)
					token = mergedToken
					continue
				}
				goto CheckExecutionCompletion
			}

			if len(outgoing) > 1 {
				// Parallel Split
				log.Info().
					Str("executionID", execution.ID).
					Str("gatewayID", currentNode.ID).
					Int("branchesCreated", len(outgoing)).
					Msg("🌿 Parallel Split Gateway creating concurrent execution branches")

				token.Status = models.ExecutionTokenStatusCompleted
				tokenRepo.UpdateExecutionToken(token, now)

				allQs, _ := queueRepo.GetQueuesByWorkflowDefinitionID(def.ID, now)
				var execQ *models.Queue
				for i := range allQs {
					if allQs[i].Type == models.WorkflowExecutionQueue {
						execQ = &allQs[i]
						break
					}
				}
				if execQ != nil && execQ.DesiredPriorityThresholds == nil {
					execQ.DesiredPriorityThresholds = map[int]int{0: 0}
					execQ.PriorityThresholds = map[int]int{0: 0}
					queueRepo.UpdateQueue(execQ, now)
				}

				for _, flow := range outgoing {
					childTokenID := strings.ReplaceAll(uuid.New().String(), "-", "")
					childToken := &models.ExecutionToken{
						ID:                   childTokenID,
						WorkflowExecutionID: execution.ID,
						WorkflowDefinitionID: def.ID,
						VNamespace:           execution.VNamespace,
						CurrentNodeID:        flow.TargetRef,
						Status:               models.ExecutionTokenStatusActive,
						ParentTokenID:        token.ID,
						CreatedAt:            now,
						UpdatedAt:            now,
					}
					tokenRepo.CreateExecutionToken(childToken, now)

					log.Info().
						Str("executionID", execution.ID).
						Str("gatewayID", currentNode.ID).
						Str("childTokenID", childToken.ID).
						Str("targetNodeID", flow.TargetRef).
						Msg("🚀 Spawned parallel branch token")

					if execQ != nil {
						msgID := strings.ReplaceAll(uuid.New().String(), "-", "")
						enqueueCmd := &queue.EnqueueCommand{
							Messages: []models.QueueMessage{
								{
									ID:          msgID,
									MessageID:   msgID,
									QueueID:     execQ.ID,
									VNamespace:  execution.VNamespace,
									Content:     []byte(fmt.Sprintf(`{"executionId":"%s","tokenId":"%s"}`, execution.ID, childToken.ID)),
									ContentType: "application/json",
									CreatedAt:   now,
									UpdatedAt:   now,
								},
							},
							CF:  cmd.CF,
							CFS: cmd.CFS,
						}
						enqueueCmd.Execute(uow, now)
					}
				}
				goto SaveExecutionState
			} else if len(outgoing) == 1 {
				token.CurrentNodeID = outgoing[0].TargetRef
				tokenRepo.UpdateExecutionToken(token, now)
			} else {
				token.Status = models.ExecutionTokenStatusCompleted
				tokenRepo.UpdateExecutionToken(token, now)
				goto CheckExecutionCompletion
			}

		case bpmn.ElementInclusiveGateway:
			outgoing := bpmnModel.GetOutgoingFlows(currentNode.ID)
			log.Info().
				Str("executionID", execution.ID).
				Str("tokenID", token.ID).
				Str("gatewayID", currentNode.ID).
				Str("gatewayName", currentNode.Name).
				Str("gatewayType", "InclusiveGateway").
				Int("outgoingFlowsCount", len(outgoing)).
				Str("defaultFlowID", currentNode.DefaultFlowID).
				Msg("🔀 Evaluating Inclusive Gateway")

			var selectedFlows []*bpmn.SequenceFlow
			var defaultFlowCandidate *bpmn.SequenceFlow

			if len(outgoing) > 1 {
				for _, flow := range outgoing {
					isDefault := currentNode.DefaultFlowID != "" && flow.ID == currentNode.DefaultFlowID
					condStr := strings.TrimSpace(flow.Condition)
					if condStr == "" && !isDefault {
						errMsg := fmt.Sprintf("gateway configuration error for %s (%s): outgoing flow %s has no condition and is not marked as default flow", currentNode.ID, currentNode.Name, flow.ID)
						log.Error().
							Str("executionID", execution.ID).
							Str("gatewayID", currentNode.ID).
							Str("flowID", flow.ID).
							Msg("❌ Gateway error: outgoing flow without condition is not marked as default flow")

						token.Status = models.ExecutionTokenStatusCancelled
						tokenRepo.UpdateExecutionToken(token, now)

						execution.Status = models.WorkflowExecutionStatusFailed
						execution.Error = errMsg
						execution.CompletedAt = &now
						execRepo.UpdateWorkflowExecution(execution, now)

						commandResult.Error = errMsg
						commandResult.Result = execution
						return *commandResult
					}
				}
			}

			for idx, flow := range outgoing {
				if currentNode.DefaultFlowID != "" && flow.ID == currentNode.DefaultFlowID {
					defaultFlowCandidate = flow
					continue
				}

				condStr := strings.TrimSpace(flow.Condition)
				matched, err := bpmn.EvaluateConditionWithResolver(condStr, execution.StateData, dbResolver)

				if err != nil {
					log.Warn().
						Err(err).
						Str("executionID", execution.ID).
						Str("gatewayID", currentNode.ID).
						Str("flowID", flow.ID).
						Str("targetRef", flow.TargetRef).
						Str("condition", condStr).
						Msg("⚠️ Inclusive Gateway condition evaluation error")
				} else {
					log.Info().
						Str("executionID", execution.ID).
						Str("gatewayID", currentNode.ID).
						Str("flowID", flow.ID).
						Str("targetRef", flow.TargetRef).
						Str("condition", condStr).
						Bool("evaluationPassed", matched).
						Int("flowIndex", idx).
						Msg("🔍 Evaluated inclusive flow condition")
				}

				if err == nil && matched {
					selectedFlows = append(selectedFlows, flow)
				}
			}

			if len(selectedFlows) == 0 && defaultFlowCandidate != nil {
				selectedFlows = append(selectedFlows, defaultFlowCandidate)
				log.Info().
					Str("executionID", execution.ID).
					Str("gatewayID", currentNode.ID).
					Str("defaultFlowID", defaultFlowCandidate.ID).
					Str("targetRef", defaultFlowCandidate.TargetRef).
					Msg("↩️ Inclusive Gateway: No conditions satisfied. Taking default flow path")
			}

			if len(selectedFlows) == 0 {
				errMsg := fmt.Sprintf("inclusive gateway evaluation failed for %s (%s): no condition satisfied and no default flow defined", currentNode.ID, currentNode.Name)
				log.Error().
					Str("executionID", execution.ID).
					Str("gatewayID", currentNode.ID).
					Str("gatewayName", currentNode.Name).
					Msg("❌ Inclusive Gateway evaluation failed: no condition met and no default flow available")

				token.Status = models.ExecutionTokenStatusCancelled
				tokenRepo.UpdateExecutionToken(token, now)

				execution.Status = models.WorkflowExecutionStatusFailed
				execution.Error = errMsg
				execution.CompletedAt = &now
				execRepo.UpdateWorkflowExecution(execution, now)

				commandResult.Error = errMsg
				commandResult.Result = execution
				return *commandResult
			}

			log.Info().
				Str("executionID", execution.ID).
				Str("gatewayID", currentNode.ID).
				Int("selectedBranches", len(selectedFlows)).
				Msg("✅ Inclusive Gateway paths selected successfully")

			if len(selectedFlows) == 1 {
				token.CurrentNodeID = selectedFlows[0].TargetRef
				tokenRepo.UpdateExecutionToken(token, now)
			} else {
				token.Status = models.ExecutionTokenStatusCompleted
				tokenRepo.UpdateExecutionToken(token, now)

				allQs, _ := queueRepo.GetQueuesByWorkflowDefinitionID(def.ID, now)
				var execQ *models.Queue
				for i := range allQs {
					if allQs[i].Type == models.WorkflowExecutionQueue {
						execQ = &allQs[i]
						break
					}
				}

				for _, flow := range selectedFlows {
					childTokenID := strings.ReplaceAll(uuid.New().String(), "-", "")
					childToken := &models.ExecutionToken{
						ID:                   childTokenID,
						WorkflowExecutionID: execution.ID,
						WorkflowDefinitionID: def.ID,
						VNamespace:           execution.VNamespace,
						CurrentNodeID:        flow.TargetRef,
						Status:               models.ExecutionTokenStatusActive,
						ParentTokenID:        token.ID,
						CreatedAt:            now,
						UpdatedAt:            now,
					}
					tokenRepo.CreateExecutionToken(childToken, now)

					if execQ != nil {
						msgID := strings.ReplaceAll(uuid.New().String(), "-", "")
						enqueueCmd := &queue.EnqueueCommand{
							Messages: []models.QueueMessage{
								{
									ID:          msgID,
									MessageID:   msgID,
									QueueID:     execQ.ID,
									VNamespace:  execution.VNamespace,
									Content:     []byte(fmt.Sprintf(`{"executionId":"%s","tokenId":"%s"}`, execution.ID, childToken.ID)),
									ContentType: "application/json",
									CreatedAt:   now,
									UpdatedAt:   now,
								},
							},
							CF:  cmd.CF,
							CFS: cmd.CFS,
						}
						enqueueCmd.Execute(uow, now)
					}
				}
				goto SaveExecutionState
			}

		case bpmn.ElementServiceTask, bpmn.ElementUserTask, bpmn.ElementTask, bpmn.ElementScriptTask:
			// Check if a job for this token & activity already exists or has completed
			existingJobs, _ := jobRepo.GetJobsByExecutionID(execution.ID, now)
			var completedJob *models.WorkflowJob
			var activeJob *models.WorkflowJob
			for i := range existingJobs {
				j := &existingJobs[i]
				if j.ExecutionTokenID == token.ID && j.ActivityID == currentNode.ID {
					if j.Status == models.WorkflowJobStatusCompleted {
						completedJob = j
						break
					} else if j.Status == models.WorkflowJobStatusPending || j.Status == models.WorkflowJobStatusAssigned {
						activeJob = j
					}
				}
			}

			if completedJob != nil {
				outgoing := bpmnModel.GetOutgoingFlows(currentNode.ID)
				if len(outgoing) > 0 {
					token.CurrentNodeID = outgoing[0].TargetRef
					tokenRepo.UpdateExecutionToken(token, now)
					continue
				} else {
					token.Status = models.ExecutionTokenStatusCompleted
					tokenRepo.UpdateExecutionToken(token, now)
					goto CheckExecutionCompletion
				}
			}

			if activeJob != nil {
				token.Status = models.ExecutionTokenStatusWaiting
				tokenRepo.UpdateExecutionToken(token, now)
				goto SaveExecutionState
			}

			jobID := strings.ReplaceAll(uuid.New().String(), "-", "")
			rawActType := currentNode.Properties["taskType"]
			if rawActType == "" {
				rawActType = currentNode.Properties["modelerTemplate"]
			}
			if rawActType == "" {
				rawActType = string(currentNode.Type)
			}
			actType := rawActType

			jobInputPayload := make(map[string]interface{})
			if execution.StateData != nil {
				for k, v := range execution.StateData {
					jobInputPayload[k] = v
				}
			}
			if currentNode.Properties != nil {
				for k, v := range currentNode.Properties {
					if k != "modelerTemplate" && k != "taskType" {
						jobInputPayload[k] = v
					}
				}
			}

			// If rawActType references a custom ActivityTemplate, resolve its RootActivity and merge its inherited properties
			// before evaluating ${...} expressions.
			if !db.IsBuiltinActivityType(rawActType) {
				jobInputPayload["_templateCode"] = rawActType

				primaryTplRepo, _ := db.NewActivityTemplateRepository(uow, idFactory, cmd.CF, cmd.CFS)
				var globalTplRepo *db.ActivityTemplateRepository
				if cmd.CF != db.AdminFC || cmd.CFS != db.AdminFCSector {
					globalTplRepo, _ = db.NewActivityTemplateRepository(uow, idFactory, db.AdminFC, db.AdminFCSector)
				}

				resolveSingleTpl := func(codeOrID string) *models.ActivityTemplate {
					if primaryTplRepo != nil {
						if found, _ := primaryTplRepo.ResolveByCodeOrID(codeOrID, execution.VNamespace, now); found != nil {
							return found
						}
					}
					if globalTplRepo != nil {
						if found, _ := globalTplRepo.ResolveByCodeOrID(codeOrID, execution.VNamespace, now); found != nil {
							return found
						}
					}
					return nil
				}

				var chainRootToLeaf []*models.ActivityTemplate
				visited := make(map[string]bool)
				currCode := strings.TrimSpace(rawActType)
				resolvedRoot := ""

				for depth := 0; depth < 10 && currCode != ""; depth++ {
					lower := strings.ToLower(currCode)
					if visited[lower] {
						break
					}
					visited[lower] = true

					if canonical := db.NormalizeBuiltinActivityType(currCode); canonical != "" {
						if resolvedRoot == "" {
							resolvedRoot = canonical
						}
						break
					}

					tpl := resolveSingleTpl(currCode)
					if tpl == nil {
						break
					}

					chainRootToLeaf = append([]*models.ActivityTemplate{tpl}, chainRootToLeaf...)
					if resolvedRoot == "" {
						if canonicalRoot := db.NormalizeBuiltinActivityType(tpl.RootActivity); canonicalRoot != "" {
							resolvedRoot = canonicalRoot
						}
					}

					nextParent := strings.TrimSpace(tpl.ParentTemplateId)
					if nextParent == "" || db.IsBuiltinActivityType(nextParent) {
						if resolvedRoot == "" {
							if canonicalParent := db.NormalizeBuiltinActivityType(nextParent); canonicalParent != "" {
								resolvedRoot = canonicalParent
							} else if inferred := db.InferBaseActivityTypeFromPayload(tpl.Payload); inferred != "" {
								resolvedRoot = inferred
							}
						}
						break
					}
					currCode = nextParent
				}

				if len(chainRootToLeaf) > 0 {
					jobInputPayload = db.MergeActivityTemplateHierarchy(chainRootToLeaf, jobInputPayload)
					if resolvedRoot == "" {
						if inferred := db.InferBaseActivityTypeFromPayload(chainRootToLeaf[0].Payload); inferred != "" {
							resolvedRoot = inferred
						}
					}
				}
				if resolvedRoot != "" {
					actType = resolvedRoot
				}
			}

			// Core Execution Interceptor: Dynamically resolve all ${variableName} expressions in task properties/payload
			evaluatedPayload, err := bpmn.EvaluateObjectResolvable(jobInputPayload, execution.StateData, dbResolver)
			if err != nil {
				errMsg := fmt.Sprintf("expression evaluation failed for node %s (%s): %v", currentNode.ID, currentNode.Name, err)
				log.Error().
					Err(err).
					Str("executionID", execution.ID).
					Str("nodeID", currentNode.ID).
					Msg("❌ Expression evaluation failed for node")

				token.Status = models.ExecutionTokenStatusCancelled
				tokenRepo.UpdateExecutionToken(token, now)

				execution.Status = models.WorkflowExecutionStatusFailed
				execution.Error = errMsg
				execution.CompletedAt = &now
				execRepo.UpdateWorkflowExecution(execution, now)

				commandResult.Error = errMsg
				commandResult.Result = execution
				return *commandResult
			}
			if evaluatedMap, ok := evaluatedPayload.(map[string]interface{}); ok {
				jobInputPayload = evaluatedMap
			}

			job := &models.WorkflowJob{
				ID:                   jobID,
				WorkflowExecutionID: execution.ID,
				ExecutionTokenID:     token.ID,
				WorkflowDefinitionID: def.ID,
				VNamespace:           execution.VNamespace,
				ActivityID:           currentNode.ID,
				ActivityName:         currentNode.Name,
				ActivityType:         actType,
				Status:               models.WorkflowJobStatusPending,
				Input:                jobInputPayload,
				Retries:              0,
				MaxRetries:           3,
				TimeoutSeconds:       300,
				CreatedAt:            now,
				UpdatedAt:            now,
			}
			if _, err := jobRepo.CreateWorkflowJob(job, now); err != nil {
				commandResult.Error = fmt.Sprintf("failed to create workflow job: %v", err)
				return *commandResult
			}

			token.Status = models.ExecutionTokenStatusWaiting
			tokenRepo.UpdateExecutionToken(token, now)

			// Enqueue Job into Activity Queue
			allQs, _ := queueRepo.GetQueuesByWorkflowDefinitionID(def.ID, now)
			var actQ *models.Queue
			for i := range allQs {
				if allQs[i].Type == models.WorkflowActivityQueue {
					actQ = &allQs[i]
					break
				}
			}

			if actQ == nil {
				// Fallback: Search queue by convention code
				actCode := fmt.Sprintf("wf-act-%s", def.Code)
				if qByCode, _ := queueRepo.GetQueueByCode(actCode, execution.VNamespace, now); qByCode != nil {
					actQ = qByCode
				} else if qByCodeDef, _ := queueRepo.GetQueueByCode(actCode, "default", now); qByCodeDef != nil {
					actQ = qByCodeDef
				}
			}

			if actQ == nil {
				errMsg := fmt.Sprintf("activity queue not found for workflow definition %s (%s)", def.ID, def.Code)
				log.Error().Str("executionID", execution.ID).Str("nodeID", currentNode.ID).Msg(errMsg)
				token.Status = models.ExecutionTokenStatusCancelled
				tokenRepo.UpdateExecutionToken(token, now)

				execution.Status = models.WorkflowExecutionStatusFailed
				execution.Error = errMsg
				execution.CompletedAt = &now
				execRepo.UpdateWorkflowExecution(execution, now)

				commandResult.Error = errMsg
				commandResult.Result = execution
				return *commandResult
			}

			if actQ.Type != models.WorkflowActivityQueue || actQ.WorkflowDefinitionID != def.ID {
				actQ.Type = models.WorkflowActivityQueue
				actQ.WorkflowDefinitionID = def.ID
				queueRepo.UpdateQueue(actQ, now)
			}

			if actQ.DesiredPriorityThresholds == nil {
				actQ.DesiredPriorityThresholds = map[int]int{0: 0}
				actQ.PriorityThresholds = map[int]int{0: 0}
				queueRepo.UpdateQueue(actQ, now)
			}

			msgID := strings.ReplaceAll(uuid.New().String(), "-", "")
			contentBytes, _ := json.Marshal(map[string]interface{}{
				"executionId":  execution.ID,
				"tokenId":      token.ID,
				"jobId":        job.ID,
				"activityId":   currentNode.ID,
				"activityName": currentNode.Name,
				"activityType": actType,
				"input":        jobInputPayload,
			})
			enqueueCmd := &queue.EnqueueCommand{
				Messages: []models.QueueMessage{
					{
						ID:          msgID,
						MessageID:   msgID,
						QueueID:     actQ.ID,
						VNamespace:  execution.VNamespace,
						Content:     contentBytes,
						ContentType: "application/json",
						CreatedAt:   now,
						UpdatedAt:   now,
					},
				},
				CF:  cmd.CF,
				CFS: cmd.CFS,
			}
			enqRes := enqueueCmd.Execute(uow, now)
			if enqRes.Error != "" {
				errMsg := fmt.Sprintf("failed to enqueue activity job message: %s", enqRes.Error)
				log.Error().Err(fmt.Errorf("%s", enqRes.Error)).Str("executionID", execution.ID).Str("jobID", job.ID).Msg(errMsg)
				token.Status = models.ExecutionTokenStatusCancelled
				tokenRepo.UpdateExecutionToken(token, now)

				execution.Status = models.WorkflowExecutionStatusFailed
				execution.Error = errMsg
				execution.CompletedAt = &now
				execRepo.UpdateWorkflowExecution(execution, now)

				commandResult.Error = errMsg
				commandResult.Result = execution
				return *commandResult
			}

			// Stop advancing this token until job completes
			goto SaveExecutionState

		case bpmn.ElementIntermediateCatchEvent, bpmn.ElementIntermediateThrowEvent, bpmn.ElementBoundaryEvent:
			rawDuration := currentNode.Properties["timeDuration"]
			rawDate := currentNode.Properties["timeDate"]
			rawCycle := currentNode.Properties["timeCycle"]

			if rawDuration == "" && rawDate == "" && rawCycle == "" {
				// Untyped intermediate event: pass through to outgoing flow
				outgoing := bpmnModel.GetOutgoingFlows(currentNode.ID)
				if len(outgoing) > 0 {
					token.CurrentNodeID = outgoing[0].TargetRef
					tokenRepo.UpdateExecutionToken(token, now)
					continue
				} else {
					token.Status = models.ExecutionTokenStatusCompleted
					tokenRepo.UpdateExecutionToken(token, now)
					goto CheckExecutionCompletion
				}
			}

			// If token was already put into waiting state for this timer, its scheduled time has elapsed
			if wasWaitingToken {
				wasWaitingToken = false
				log.Info().
					Str("executionID", execution.ID).
					Str("tokenID", token.ID).
					Str("nodeID", currentNode.ID).
					Msg("✅ Scheduled timer elapsed! Token resuming node and advancing to next flow target")

				outgoing := bpmnModel.GetOutgoingFlows(currentNode.ID)
				if len(outgoing) > 0 {
					token.CurrentNodeID = outgoing[0].TargetRef
					token.Status = models.ExecutionTokenStatusActive
					tokenRepo.UpdateExecutionToken(token, now)
					continue
				} else {
					token.Status = models.ExecutionTokenStatusCompleted
					tokenRepo.UpdateExecutionToken(token, now)
					goto CheckExecutionCompletion
				}
			}

			// Evaluate ${...} expressions if present in timer values
			evaluatedDurationStr := rawDuration
			if strings.Contains(rawDuration, "${") {
				if res, err := bpmn.EvaluateStringResolvable(rawDuration, execution.StateData, dbResolver); err == nil && res != nil {
					evaluatedDurationStr = fmt.Sprintf("%v", res)
				}
			}

			evaluatedDateStr := rawDate
			if strings.Contains(rawDate, "${") {
				if res, err := bpmn.EvaluateStringResolvable(rawDate, execution.StateData, dbResolver); err == nil && res != nil {
					evaluatedDateStr = fmt.Sprintf("%v", res)
				}
			}

			var targetRunAt time.Time
			var calcErr error

			if strings.TrimSpace(evaluatedDurationStr) != "" {
				dur, err := bpmn.ParseHumanDuration(evaluatedDurationStr)
				if err != nil {
					calcErr = fmt.Errorf("failed to parse timer duration '%s' for node %s (%s): %w", evaluatedDurationStr, currentNode.ID, currentNode.Name, err)
				} else {
					targetRunAt = now.Add(dur)
				}
			} else if strings.TrimSpace(evaluatedDateStr) != "" {
				parsedDate, err := bpmn.ParseDateTime(evaluatedDateStr)
				if err != nil {
					calcErr = fmt.Errorf("failed to parse timer date '%s' for node %s (%s): %w", evaluatedDateStr, currentNode.ID, currentNode.Name, err)
				} else {
					targetRunAt = parsedDate
				}
			}

			if calcErr != nil {
				log.Error().Err(calcErr).Str("executionID", execution.ID).Str("nodeID", currentNode.ID).Msg("❌ Timer calculation error")
				token.Status = models.ExecutionTokenStatusCancelled
				tokenRepo.UpdateExecutionToken(token, now)

				execution.Status = models.WorkflowExecutionStatusFailed
				execution.Error = calcErr.Error()
				execution.CompletedAt = &now
				execRepo.UpdateWorkflowExecution(execution, now)

				commandResult.Error = calcErr.Error()
				commandResult.Result = execution
				return *commandResult
			}

			// Check if timer target run time is in the future
			if now.Before(targetRunAt) {
				// Put token in waiting state
				token.Status = models.ExecutionTokenStatusWaiting
				tokenRepo.UpdateExecutionToken(token, now)

				// Find or resolve workflow execution queue
				allQs, _ := queueRepo.GetQueuesByWorkflowDefinitionID(def.ID, now)
				var execQ *models.Queue
				for i := range allQs {
					if allQs[i].Type == models.WorkflowExecutionQueue {
						execQ = &allQs[i]
						break
					}
				}

				if execQ == nil {
					execCode := fmt.Sprintf("wf-exec-%s", def.Code)
					if qByCode, _ := queueRepo.GetQueueByCode(execCode, execution.VNamespace, now); qByCode != nil {
						execQ = qByCode
					} else if qByCodeDef, _ := queueRepo.GetQueueByCode(execCode, "default", now); qByCodeDef != nil {
						execQ = qByCodeDef
					}
				}

				if execQ != nil && (execQ.Type != models.WorkflowExecutionQueue || execQ.WorkflowDefinitionID != def.ID) {
					execQ.Type = models.WorkflowExecutionQueue
					execQ.WorkflowDefinitionID = def.ID
					queueRepo.UpdateQueue(execQ, now)
				}

				if execQ == nil {
					errMsg := fmt.Sprintf("workflow execution queue not found for definition %s", def.ID)
					log.Error().Str("executionID", execution.ID).Str("nodeID", currentNode.ID).Msg(errMsg)
					token.Status = models.ExecutionTokenStatusCancelled
					tokenRepo.UpdateExecutionToken(token, now)

					execution.Status = models.WorkflowExecutionStatusFailed
					execution.Error = errMsg
					execution.CompletedAt = &now
					execRepo.UpdateWorkflowExecution(execution, now)

					commandResult.Error = errMsg
					commandResult.Result = execution
					return *commandResult
				}

				// Create a One-Off ScheduledJob to resume execution when targetRunAt arrives
				payloadBytes, _ := json.Marshal(map[string]interface{}{
					"executionId": execution.ID,
					"tokenId":     token.ID,
				})

				scheduledJobRepo, errSJ := db.NewScheduledJobRepository(uow, idFactory, cmd.CF, cmd.CFS)
				if errSJ != nil {
					commandResult.Error = errSJ.Error()
					return *commandResult
				}

				schedJobID := fmt.Sprintf("timer_%s_%s_%d", execution.ID, token.ID, targetRunAt.Unix())
				scheduledJob := models.ScheduledJob{
					ID:          schedJobID,
					Code:        schedJobID,
					TenantID:    execution.VNamespace,
					VNamespace:  execution.VNamespace,
					TargetType:  string(models.ScheduledJobTargetQueue),
					TargetID:    execQ.ID,
					Type:        models.ScheduledJobOneOff,
					State:       models.ScheduledJobIdle,
					NextRunAt:   targetRunAt,
					Content:     string(payloadBytes),
					ContentType: "application/json",
					CreatedAt:   now,
					UpdatedAt:   now,
				}

				_, errCreateSJ := scheduledJobRepo.CreateScheduledJob(&scheduledJob, now)
				if errCreateSJ != nil {
					log.Warn().Err(errCreateSJ).Str("executionID", execution.ID).Str("scheduledJobID", schedJobID).Msg("Scheduled job creation notice")
				}

				log.Info().
					Str("executionID", execution.ID).
					Str("tokenID", token.ID).
					Str("nodeID", currentNode.ID).
					Str("nodeName", currentNode.Name).
					Time("targetRunAt", targetRunAt).
					Str("scheduledJobID", schedJobID).
					Msg("⏱️ Timer scheduled via queue ScheduledJob. Token put into waiting state")

				goto SaveExecutionState
			}

			// Timer has expired / due (now >= targetRunAt)
			log.Info().
				Str("executionID", execution.ID).
				Str("tokenID", token.ID).
				Str("nodeID", currentNode.ID).
				Time("targetRunAt", targetRunAt).
				Msg("✅ Timer elapsed! Token resolving node and advancing to next flow target")

			outgoing := bpmnModel.GetOutgoingFlows(currentNode.ID)
			if len(outgoing) > 0 {
				token.CurrentNodeID = outgoing[0].TargetRef
				token.Status = models.ExecutionTokenStatusActive
				tokenRepo.UpdateExecutionToken(token, now)
				continue
			} else {
				token.Status = models.ExecutionTokenStatusCompleted
				tokenRepo.UpdateExecutionToken(token, now)
				goto CheckExecutionCompletion
			}

		case bpmn.ElementEndEvent:
			token.Status = models.ExecutionTokenStatusCompleted
			tokenRepo.UpdateExecutionToken(token, now)
			goto CheckExecutionCompletion

		default:
			// Fallback: move along outgoing flow if available
			outgoing := bpmnModel.GetOutgoingFlows(currentNode.ID)
			if len(outgoing) > 0 {
				token.CurrentNodeID = outgoing[0].TargetRef
				tokenRepo.UpdateExecutionToken(token, now)
			} else {
				token.Status = models.ExecutionTokenStatusCompleted
				tokenRepo.UpdateExecutionToken(token, now)
				goto CheckExecutionCompletion
			}
		}
	}

CheckExecutionCompletion:
	{
		allTokens, _ := tokenRepo.GetTokensByExecutionID(execution.ID, now)
		hasUnfinishedTokens := false
		for _, t := range allTokens {
			if t.Status == models.ExecutionTokenStatusActive || t.Status == models.ExecutionTokenStatusWaiting {
				hasUnfinishedTokens = true
				break
			}
		}
		if !hasUnfinishedTokens {
			execution.Status = models.WorkflowExecutionStatusCompleted
			execution.CompletedAt = &now
			execution.Output = execution.StateData
		}
	}

SaveExecutionState:
	execRepo.UpdateWorkflowExecution(execution, now)
	commandResult.Result = execution
	return *commandResult
}

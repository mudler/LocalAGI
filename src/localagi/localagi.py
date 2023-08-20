import os
import openai
import requests
from loguru import logger
import json

DEFAULT_API_BASE = "http://api:8080"
VOICE_MODEL = "en-us-kathleen-low.onnx"
STABLEDIFFUSION_MODEL = "stablediffusion"
FUNCTIONS_MODEL = "functions"
EMBEDDINGS_MODEL = "all-MiniLM-L6-v2"
LLM_MODEL = "gpt-4"

# LocalAGI class
class LocalAGI:
    # Constructor
    def __init__(self, 
                 plan_action="plan", 
                 reply_action="reply",
                 force_action="",
                 agent_actions={}, 
                 plan_message="",
                 api_base=DEFAULT_API_BASE, 
                 tts_api_base="", 
                 stablediffusion_api_base="",
                 embeddings_api_base="", 
                 tts_model=VOICE_MODEL, 
                 stablediffusion_model=STABLEDIFFUSION_MODEL, 
                 functions_model=FUNCTIONS_MODEL, 
                 embeddings_model=EMBEDDINGS_MODEL, 
                 llm_model=LLM_MODEL,
                 tts_player="aplay",
                 ):
        self.api_base = api_base
        self.agent_actions = agent_actions
        self.plan_message = plan_message
        self.force_action = force_action
        self.processed_messages=0
        self.tts_player = tts_player
        self.agent_actions[plan_action] = {
                                            "function": self.generate_plan,
                                            "plannable": False,
                                            "description": 'The assistant for solving complex tasks that involves calling more functions in sequence, replies with the action "'+plan_action+'".',
                                            "signature": {
                                                "name": plan_action,
                                                "description": """Plan complex tasks.""",
                                                "parameters": {
                                                    "type": "object",
                                                    "properties": {
                                                        "description": {
                                                            "type": "string",
                                                            "description": "reasoning behind the planning"
                                                        },
                                                    },
                                                    "required": ["description"]
                                                }
                                            },
                                        }
        self.agent_actions[reply_action] = {
                                        "function": None,
                                        "plannable": False,
                                        "description": 'For replying to the user, the assistant replies with the action "'+reply_action+'" and the reply to the user directly when there is nothing to do.',
                                    }
        self.tts_api_base = tts_api_base if tts_api_base else self.api_base
        self.stablediffusion_api_base = stablediffusion_api_base if stablediffusion_api_base else self.api_base
        self.embeddings_api_base = embeddings_api_base if embeddings_api_base else self.api_base
        self.tts_model = tts_model
        self.stablediffusion_model = stablediffusion_model
        self.functions_model = functions_model
        self.embeddings_model = embeddings_model
        self.llm_model = llm_model
        self.reply_action = reply_action
    # Function to create images with LocalAI
    def get_avatar(self, input_text):
        response = openai.Image.create(
            prompt=input_text,
            n=1,
            size="128x128",
            api_base=self.sta+"/v1"
        )
        return response['data'][0]['url']

    def tts_play(self, input_text):
        output_file_path = '/tmp/output.wav'
        self.tts(input_text, output_file_path)
        try:
            # Use aplay to play the audio
            os.system(f"{self.tts_player} {output_file_path}")
            # remove the audio file
            os.remove(output_file_path)
        except:
            logger.info('Unable to play audio')
        
    # Function to create audio with LocalAI
    def tts(self, input_text, output_file_path):
        # strip newlines from text
        input_text = input_text.replace("\n", ".")

        # get from OPENAI_API_BASE env var
        url = self.tts_api_base + '/tts'
        headers = {'Content-Type': 'application/json'}
        data = {
            "input": input_text,
            "model": self.tts_model,
        }

        response = requests.post(url, headers=headers, data=json.dumps(data))

        if response.status_code == 200:
            with open(output_file_path, 'wb') as f:
                f.write(response.content)
            logger.info('Audio file saved successfully:', output_file_path)
        else:
            logger.info('Request failed with status code', response.status_code)

    # Function to analyze the user input and pick the next action to do
    def needs_to_do_action(self, user_input, agent_actions={}):
        if len(agent_actions) == 0:
            agent_actions = self.agent_actions
        # Get the descriptions and the actions name (the keys)
        descriptions=self.action_description("", agent_actions)

        messages = [
                {"role": "user",
                "content": f"""Transcript of AI assistant responding to user requests. Replies with the action to perform and the reasoning.
    {descriptions}"""},
                {"role": "user",
    "content": f"""{user_input}
    Function call: """
                }
            ]
        functions = [
            {
            "name": "intent",
            "description": """Decide to do an action.""",
            "parameters": {
                "type": "object",
                "properties": {
                "confidence": {
                    "type": "number",
                    "description": "confidence of the action"
                },
                "detailed_reasoning": {
                    "type": "string",
                    "description": "reasoning behind the intent"
                },
                # "detailed_reasoning": {
                #     "type": "string",
                #     "description": "reasoning behind the intent"
                # },
                "action": {
                    "type": "string",
                    "enum": list(agent_actions.keys()),
                    "description": "user intent"
                },
                },
                "required": ["action"]
            }
            },    
        ]
        response = openai.ChatCompletion.create(
            #model="gpt-3.5-turbo",
            model=self.functions_model,
            messages=messages,
            request_timeout=1200,
            functions=functions,
            api_base=self.api_base+"/v1",
            stop=None,
            temperature=0.1,
            #function_call="auto"
            function_call={"name": "intent"},
        )
        response_message = response["choices"][0]["message"]
        if response_message.get("function_call"):
            function_name = response.choices[0].message["function_call"].name
            function_parameters = response.choices[0].message["function_call"].arguments
            # read the json from the string
            res = json.loads(function_parameters)
            logger.debug(">>> function name: "+function_name)
            logger.debug(">>> function parameters: "+function_parameters)
            return res
        return {"action": self.reply_action}

    # This is used to collect the descriptions of the agent actions, used to populate the LLM prompt
    def action_description(self, action, agent_actions):
        descriptions=""
        # generate descriptions of actions that the agent can pick
        for a in agent_actions:
            if ( action != "" and action == a ) or (action == ""):
                descriptions+=agent_actions[a]["description"]+"\n"
        return descriptions


    ### This function is used to process the functions given a user input.
    ### It picks a function, executes it and returns the list of messages containing the result.
    def process_functions(self, user_input, action="",):

        descriptions=self.action_description(action, self.agent_actions)

        messages = [
            #   {"role": "system", "content": "You are a helpful assistant."},
                {"role": "user",
                "content": f"""Transcript of AI assistant responding to user requests. Replies with the action to perform, including reasoning, and the confidence interval from 0 to 100.
    {descriptions}"""},
                {"role": "user",
    "content": f"""{user_input}
    Function call: """
                }
            ]
        response = self.function_completion(messages, action=action)
        response_message = response["choices"][0]["message"]
        response_result = ""
        function_result = {}
        if response_message.get("function_call"):
            function_name = response.choices[0].message["function_call"].name
            function_parameters = response.choices[0].message["function_call"].arguments
            logger.info("==> function parameters: {function_parameters}",function_parameters=function_parameters)
            function_to_call = self.agent_actions[function_name]["function"]

            function_result = function_to_call(function_parameters, agent_actions=self.agent_actions, localagi=self)
            logger.info("==> function result: {function_result}", function_result=function_result)
            messages.append(
                {
                    "role": "assistant",
                    "content": None,
                    "function_call": {"name": function_name, "arguments": function_parameters,},
                }
            )
            messages.append(
                {
                    "role": "function",
                    "name": function_name,
                    "content": str(function_result)
                }
            )
        return messages, function_result

    ### function_completion is used to autocomplete functions given a list of messages
    def function_completion(self, messages, action=""):
        function_call = "auto"
        if action != "":
            function_call={"name": action}
        logger.debug("==> function name: {function_call}", function_call=function_call)
        # get the functions from the signatures of the agent actions, if exists
        functions = []
        for action in self.agent_actions:
            if self.agent_actions[action].get("signature"):
                functions.append(self.agent_actions[action]["signature"])
        response = openai.ChatCompletion.create(
            #model="gpt-3.5-turbo",
            model=self.functions_model,
            messages=messages,
            functions=functions,
            request_timeout=1200,
            stop=None,
            api_base=self.api_base+"/v1",
            temperature=0.1,
            function_call=function_call
        )

        return response

    # Rework the content of each message in the history in a way that is understandable by the LLM
    # TODO: switch to templates (?)
    def process_history(self, conversation_history):
        messages = ""
        for message in conversation_history:
            # if there is content append it
            if message.get("content") and message["role"] == "function":
                messages+="Function result: \n" + message["content"]+"\n"
            elif message.get("function_call"):
                # encode message["function_call" to json and appends it
                fcall = json.dumps(message["function_call"])
                parameters = "calling " + message["function_call"]["name"]+" with arguments:"
                args=json.loads(message["function_call"]["arguments"])
                for arg in args:
                    logger.debug(arg)
                    logger.debug(args)
                    v=args[arg]
                    parameters+=f""" {arg}=\"{v}\""""
                messages+= parameters+"\n"
            elif message.get("content") and message["role"] == "user":
                messages+=message["content"]+"\n"
            elif message.get("content") and message["role"] == "assistant":
                messages+="Assistant message: "+message["content"]+"\n"
        return messages

    def converse(self, responses):
        response = openai.ChatCompletion.create(
            model=self.llm_model,
            messages=responses,
            stop=None,
            api_base=self.api_base+"/v1",
            request_timeout=1200,
            temperature=0.1,
        )
        responses.append(
            {
                "role": "assistant",
                "content": response.choices[0].message["content"],
            }
        )
        return responses

    ### Fine tune a string before feeding into the LLM

    def analyze(self, responses, prefix="Analyze the following text highlighting the relevant information and identify a list of actions to take if there are any. If there are errors, suggest solutions to fix them", suffix=""):
        string = self.process_history(responses)
        messages = []

        if prefix != "":
            messages = [
                {
                "role": "user",
                "content": f"""{prefix}:

        ```
        {string}
        ```
        """,
                }
            ]
        else:
            messages = [
                {
                "role": "user",
                "content": f"""{string}""",
                }
            ]

        if suffix != "":
            messages[0]["content"]+=f"""{suffix}"""
    
        response = openai.ChatCompletion.create(
            model=self.llm_model,
            messages=messages,
            stop=None,
            api_base=self.api_base+"/v1",
            request_timeout=1200,
            temperature=0.1,
        )
        return  response.choices[0].message["content"]

    def post_process(self, string):
        messages = [
            {
            "role": "user",
            "content": f"""Summarize the following text, keeping the relevant information:

    ```
    {string}
    ```
    """,
            }
        ]
        logger.info("==> Post processing: {string}", string=string)
        # get the response from the model
        response = openai.ChatCompletion.create(
            model=self.llm_model,
            messages=messages,
            api_base=self.api_base+"/v1",
            stop=None,
            temperature=0.1,
            request_timeout=1200,
        )
        result = response["choices"][0]["message"]["content"]
        logger.info("==> Processed: {string}", string=result)
        return result

    def generate_plan(self, user_input, agent_actions={}, localagi=None):
        res = json.loads(user_input)
        logger.info("--> Calculating plan: {description}", description=res["description"])
        descriptions=self.action_description("",agent_actions)

        plan_message = "The assistant replies with a plan to answer the request with a list of subtasks with logical steps. The reasoning includes a self-contained, detailed and descriptive instruction to fullfill the task."
        if self.plan_message:
            plan_message = self.plan_message
            # plan_message = "The assistant replies with a plan of 3 steps to answer the request with a list of subtasks with logical steps. The reasoning includes a self-contained, detailed and descriptive instruction to fullfill the task."

        messages = [
                {"role": "user",
                "content": f"""Transcript of AI assistant responding to user requests. 
    {descriptions}

    Request: {plan_message}
    Thought: {res["description"]}
    Function call: """
                }
            ]
        # get list of plannable actions
        plannable_actions = []
        for action in agent_actions:
            if agent_actions[action]["plannable"]:
                # append the key of the dict to plannable_actions
                plannable_actions.append(action)

        functions = [
            {
            "name": "plan",
            "description": """Decide to do an action.""",
            "parameters": {
                "type": "object",
                "properties": {
                    "subtasks": {
                        "type": "array",
                        "items": {
                            "type": "object",
                            "properties": {
                                "detailed_reasoning": {
                                    "type": "string",
                                    "description": "subtask list",
                                },
                                "function": {
                                    "type": "string",
                                    "enum": plannable_actions,
                                },               
                            },
                        },
                    },
                },
                "required": ["subtasks"]
            }
            },    
        ]
        response = openai.ChatCompletion.create(
            #model="gpt-3.5-turbo",
            model=self.functions_model,
            messages=messages,
            functions=functions,
            api_base=self.api_base+"/v1",
            stop=None,
            temperature=0.1,
            #function_call="auto"
            function_call={"name": "plan"},
        )
        response_message = response["choices"][0]["message"]
        if response_message.get("function_call"):
            function_name = response.choices[0].message["function_call"].name
            function_parameters = response.choices[0].message["function_call"].arguments
            # read the json from the string
            res = json.loads(function_parameters)
            logger.debug("<<< function name: {function_name} >>>> parameters: {parameters}", function_name=function_name,parameters=function_parameters)
            return res
        return {"action": self.reply_action}
    
    def evaluate(self,user_input, conversation_history = [], critic=True, re_evaluate=False,re_evaluation_in_progress=False, postprocess=False, subtaskContext=False):
        messages = [
            {
            "role": "user",
            "content": user_input,
            }
        ]

        conversation_history.extend(messages)

        # pulling the old history make the context grow exponentially
        # and most importantly it repeates the first message with the commands again and again.
        # it needs a bit of cleanup and process the messages and piggyback more LocalAI functions templates
        # old_history = process_history(conversation_history)
        # action_picker_message = "Conversation history:\n"+old_history
        # action_picker_message += "\n"
        action_picker_message = "Request: "+user_input

        picker_actions = self.agent_actions
        if self.force_action:
            aa = {}
            aa[self.force_action] = self.agent_actions[self.force_action]
            picker_actions = aa
            logger.info("==> Forcing action to '{action}' as requested by the user", action=self.force_action)

        #if re_evaluate and not re_evaluation_in_progress:
        #    observation = analyze(conversation_history, prefix=True)
        #    action_picker_message+="\n\Thought: "+observation[-1]["content"]
        if re_evaluation_in_progress:
            observation = self.analyze(conversation_history)
            action_picker_message="Decide from the output below if we have to do another action:\n"
            action_picker_message+="```\n"+user_input+"\n```"
            action_picker_message+="\n\nObservation: "+observation
            # if there is no action to do, we can just reply to the user with REPLY_ACTION
        try:
            action = self.needs_to_do_action(action_picker_message,agent_actions=picker_actions)
        except Exception as e:
            logger.error("==> error: ")
            logger.error(e)
            action = {"action": self.reply_action}

        if action["action"] != self.reply_action:
            logger.info("==> LocalAGI wants to call '{action}'", action=action["action"])
            #logger.info("==> Observation '{reasoning}'", reasoning=action["detailed_reasoning"])
            logger.info("==> Reasoning '{reasoning}'", reasoning=action["detailed_reasoning"])
            # Force executing a plan instead

            reasoning = action["detailed_reasoning"]
            if action["action"] == self.reply_action:
                logger.info("==> LocalAGI wants to create a plan that involves more actions ")

            #if postprocess:
                #reasoning = post_process(reasoning)
            function_completion_message=""
            if self.processed_messages > 0:
                function_completion_message += self.process_history(conversation_history)+"\n"
            function_completion_message += "Request: "+user_input+"\nReasoning: "+reasoning

            responses, function_results = self.process_functions(function_completion_message, action=action["action"])
            # Critic re-evaluates the action
            # if critic:
            #     critic = self.analyze(responses[1:-1], suffix=f"Analyze if the function that was picked is correct and satisfies the user request from the context above. Suggest a different action if necessary. If the function picked was correct, write the picked function.\n")
            #     logger.info("==> Critic action: {critic}", critic=critic)
            #     previous_action = action["action"]
            #     try:
            #         action = self.needs_to_do_action(critic,agent_actions=picker_actions)
            #         if action["action"] != previous_action:
            #             logger.info("==> Critic decided to change action to: {action}", action=action["action"])
            #         responses, function_results = self.process_functions(function_completion_message, action=action["action"])
            #     except Exception as e:
            #         logger.error("==> error: ")
            #         logger.error(e)
            #         action = {"action": self.reply_action}

            # Critic re-evaluates the plan
            if critic and isinstance(function_results, dict) and function_results.get("subtasks") and len(function_results["subtasks"]) > 0:
                critic = self.analyze(responses[1:], prefix="", suffix=f"Analyze if the plan is correct and satisfies the user request from the context above. Suggest a revised plan if necessary.\n")
                logger.info("==> Critic plan: {critic}", critic=critic)
                responses, function_results = self.process_functions(function_completion_message+"\n"+critic, action=action["action"])

            # if there are no subtasks, we can just reply,
            # otherwise we execute the subtasks
            # First we check if it's an object
            if isinstance(function_results, dict) and function_results.get("subtasks") and len(function_results["subtasks"]) > 0:
                # cycle subtasks and execute functions
                subtask_result=""
                for subtask in function_results["subtasks"]:
                    cr="Request: "+user_input+"\nReasoning: "+action["detailed_reasoning"]+ "\n"
                    #cr="Request: "+user_input+"\n"
                    #cr=""
                    if subtask_result != "" and subtaskContext:
                        # Include cumulative results of previous subtasks
                        # TODO: this grows context, maybe we should use a different approach or summarize
                        ##if postprocess:
                        ##    cr+= "Subtask results: "+post_process(subtask_result)+"\n"
                        ##else:
                        cr+="\nAdditional context: ```\n"+subtask_result+"\n```\n"
                    subtask_reasoning = subtask["detailed_reasoning"]
                    #cr+="Reasoning: "+action["detailed_reasoning"]+ "\n"
                    cr+="\nFunction to call:" +subtask["function"]+"\n"
                    logger.info("==> subtask '{subtask}' ({reasoning})", subtask=subtask["function"], reasoning=subtask_reasoning)
                    if postprocess:
                        cr+= "Assistant: "+self.post_process(subtask_reasoning)
                    else:
                        cr+= "Assistant: "+subtask_reasoning
                    subtask_response, function_results = self.process_functions(cr, subtask["function"])
                    subtask_result+=str(function_results)+"\n"
                    # if postprocess:
                    #    subtask_result=post_process(subtask_result)
                    responses.append(subtask_response[-1])
            if re_evaluate:
                ## Better output or this infinite loops..
                logger.info("-> Re-evaluate if another action is needed")
                ## ? conversation history should go after the user_input maybe?
                re_eval = ""
                # This is probably not needed as already in the history:
                #re_eval = user_input +"\n"
                #re_eval += "Conversation history: \n"
                if postprocess:
                    re_eval+= self.post_process(self.process_history(responses[1:])) +"\n"
                else:
                    re_eval+= self.process_history(responses[1:]) +"\n"
                responses = self.evaluate(re_eval, 
                                          responses, 
                                          re_evaluate,
                                          re_evaluation_in_progress=True)

            if re_evaluation_in_progress:
                conversation_history.extend(responses)
                return conversation_history
                
            # unwrap the list of responses
            conversation_history.append(responses[-1])

            #responses = converse(responses)

            # TODO: this needs to be optimized
            responses = self.analyze(responses[1:],
                                     prefix="", 
                                     suffix=f"Return an appropriate answer given the context above\n")

            # add responses to conversation history by extending the list
            conversation_history.append(
                {
                "role": "assistant",
                "content": responses,
                }
            )

            self.processed_messages+=1
            # logger.info the latest response from the conversation history
            logger.info(conversation_history[-1]["content"])
            #self.tts(conversation_history[-1]["content"])
        else:
            logger.info("==> no action needed")

            if re_evaluation_in_progress:
                logger.info("==> LocalAGI has completed the user request")
                logger.info("==> LocalAGI will reply to the user")
                return conversation_history        

            # get the response from the model
            response = self.converse(conversation_history)
            self.processed_messages+=1

            # add the response to the conversation history by extending the list
            conversation_history.extend(response)
            # logger.info the latest response from the conversation history
            logger.info(conversation_history[-1]["content"])
            #self.tts(conversation_history[-1]["content"])
        return conversation_history
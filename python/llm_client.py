#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
大模型客户端 - 用于从Go调用大模型API
支持OpenAI、Anthropic等模型
"""

import json
import sys
import os
from typing import Dict, List, Any, Optional

try:
    import openai
    from openai import OpenAI
except ImportError:
    openai = None
    OpenAI = None

try:
    import anthropic
    from anthropic import Anthropic
except ImportError:
    anthropic = None
    Anthropic = None


def load_config() -> Dict[str, Any]:
    """从环境变量或配置文件加载配置"""
    # 这里可以从环境变量读取，或者从Go传递的配置中获取
    return {}


def call_openai_api(request: Dict[str, Any], config: Dict[str, Any]) -> Dict[str, Any]:
    """调用OpenAI API"""
    if OpenAI is None:
        return {"error": "OpenAI库未安装，请运行: pip install openai"}

    api_config = config.get("api", {})
    client = OpenAI(
        api_key=api_config.get("api_key", os.getenv("OPENAI_API_KEY", "")),
        base_url=api_config.get("base_url", "https://api.openai.com/v1")
    )

    context = request.get("context", "")
    input_text = request.get("input", "")
    
    # 构建消息
    messages = []
    if context:
        messages.append({"role": "system", "content": context})
    messages.append({"role": "user", "content": input_text})

    # 调用API
    try:
        response = client.chat.completions.create(
            model=api_config.get("model", "gpt-4"),
            messages=messages,
            temperature=api_config.get("temperature", 0.7),
            max_tokens=api_config.get("max_tokens", 2000),
            top_p=api_config.get("top_p", 1.0),
            frequency_penalty=api_config.get("frequency_penalty", 0.0),
            presence_penalty=api_config.get("presence_penalty", 0.0),
        )

        text = response.choices[0].message.content
        
        # 生成多个建议（简单实现：基于不同temperature）
        suggestions = [text]
        if len(suggestions) < 3:
            # 可以生成更多变体
            suggestions.append(text)

        return {
            "text": text,
            "suggestions": suggestions[:3]
        }
    except Exception as e:
        return {"error": f"OpenAI API调用失败: {str(e)}"}


def call_anthropic_api(request: Dict[str, Any], config: Dict[str, Any]) -> Dict[str, Any]:
    """调用Anthropic API"""
    if Anthropic is None:
        return {"error": "Anthropic库未安装，请运行: pip install anthropic"}

    api_config = config.get("api", {})
    client = Anthropic(
        api_key=api_config.get("api_key", os.getenv("ANTHROPIC_API_KEY", ""))
    )

    context = request.get("context", "")
    input_text = request.get("input", "")
    
    # 构建消息
    message = f"{context}\n\n{input_text}" if context else input_text

    try:
        response = client.messages.create(
            model=api_config.get("model", "claude-3-opus-20240229"),
            max_tokens=api_config.get("max_tokens", 2000),
            temperature=api_config.get("temperature", 0.7),
            messages=[{"role": "user", "content": message}]
        )

        text = response.content[0].text
        suggestions = [text]

        return {
            "text": text,
            "suggestions": suggestions[:3]
        }
    except Exception as e:
        return {"error": f"Anthropic API调用失败: {str(e)}"}


def generate_summary(request: Dict[str, Any], config: Dict[str, Any]) -> Dict[str, Any]:
    """生成对话摘要"""
    messages = request.get("messages", [])
    existing_summary = request.get("existing_summary")
    summary_config = request.get("config", {})

    # 构建摘要提示词
    prompt = "请分析以下对话，生成一个简洁的摘要，包含关键信息和对话主题。\n\n"
    
    if existing_summary:
        prompt += f"已有摘要：{existing_summary.get('prompt', '')}\n\n"
        prompt += "请基于新消息更新摘要。\n\n"

    # 添加消息
    prompt += "对话内容：\n"
    for msg in messages[-100:]:  # 只取最近100条消息
        prompt += f"[{msg.get('sender_id', 'unknown')}]: {msg.get('content', '')}\n"

    prompt += "\n请生成：\n1. 一个简洁的摘要提示词（用于后续对话上下文）\n2. 关键信息列表（JSON格式）"

    # 调用大模型生成摘要
    api_config = config.get("api", {})
    model_type = config.get("model_type", "openai")

    if model_type == "openai" and OpenAI:
        client = OpenAI(
            api_key=api_config.get("api_key", os.getenv("OPENAI_API_KEY", "")),
            base_url=api_config.get("base_url", "https://api.openai.com/v1")
        )
        
        try:
            response = client.chat.completions.create(
                model=api_config.get("model", "gpt-4"),
                messages=[{"role": "user", "content": prompt}],
                temperature=0.3,  # 摘要使用较低temperature
                max_tokens=summary_config.get("max_summary_tokens", 500),
            )
            
            result_text = response.choices[0].message.content
            
            # 简单解析结果（实际应该更智能）
            lines = result_text.split("\n")
            summary_prompt = ""
            key_info = []
            
            in_key_info = False
            for line in lines:
                if "摘要" in line or "提示词" in line:
                    continue
                if "关键信息" in line or "JSON" in line:
                    in_key_info = True
                    continue
                if not in_key_info:
                    summary_prompt += line + "\n"
                else:
                    # 尝试解析JSON
                    try:
                        if line.strip().startswith("["):
                            key_info = json.loads(line.strip())
                    except:
                        pass

            return {
                "prompt": summary_prompt.strip(),
                "key_info": key_info if key_info else []
            }
        except Exception as e:
            return {"error": f"生成摘要失败: {str(e)}"}
    
    return {"error": "不支持的大模型类型或库未安装"}


def handle_complete(request: Dict[str, Any], config: Dict[str, Any]) -> Dict[str, Any]:
    """处理补全请求"""
    model_type = config.get("model_type", "openai")
    
    if model_type == "openai":
        return call_openai_api(request, config)
    elif model_type == "anthropic":
        return call_anthropic_api(request, config)
    else:
        return {"error": f"不支持的大模型类型: {model_type}"}


def main():
    """主函数"""
    try:
        # 从stdin读取JSON请求
        input_data = sys.stdin.read()
        request_data = json.loads(input_data)
        
        action = request_data.get("action")
        request = request_data.get("request", {})
        config = request_data.get("config", {})
        
        if action == "complete":
            result = handle_complete(request, config)
        elif action == "generate_summary":
            result = generate_summary(request, config)
        else:
            result = {"error": f"未知的操作: {action}"}
        
        # 输出JSON结果到stdout
        print(json.dumps(result, ensure_ascii=False))
        
    except json.JSONDecodeError as e:
        error_result = {"error": f"JSON解析失败: {str(e)}"}
        print(json.dumps(error_result, ensure_ascii=False))
        sys.exit(1)
    except Exception as e:
        error_result = {"error": f"处理失败: {str(e)}"}
        print(json.dumps(error_result, ensure_ascii=False))
        sys.exit(1)


if __name__ == "__main__":
    main()


#!/usr/bin/env python3
"""从社保模板中提取下拉枚举与默认值，生成后端可用的常量 JSON。"""

from __future__ import annotations

import json
import re
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Iterable, List, Sequence, Tuple

import xlrd

ROOT = Path(__file__).resolve().parents[1]
INCREASE_TEMPLATE = ROOT / "社保缴费人员增加申报（企业职工批量新参保）模版.xls"
DECREASE_TEMPLATE = ROOT / "社保缴费人员减少申报（企业职工批量减少参保）模板.xls"
OUTPUT_JSON = ROOT / "backend" / "internal" / "api" / "data" / "social_template_constants.json"

KEYWORDS_PERSONAL_ID = (
    "公务员",
    "事业",
    "农民",
    "学生",
    "军人",
    "自由职业",
    "个体",
    "无业",
    "退",
    "工人",
    "管理人员",
    "职员",
    "其他",
)
KEYWORDS_EDUCATION = (
    "博士研究生",
    "硕士研究生",
    "大学本科",
    "大学专科",
    "中等专科",
    "职业高中",
    "技工学校",
    "普通中学",
    "初级中学",
    "小学",
)

SPECIAL_SKILL_CODES = {"410", "411", "412", "420", "430", "434", "435", "499"}
SKILL_LEVEL_CODES = {"1", "2", "3", "4", "5"}


@dataclass
class OptionSet:
    options: List[str]
    default: str

    def as_dict(self) -> dict:
        return {"options": self.options, "default": self.default}


def ensure_output_dir() -> None:
    OUTPUT_JSON.parent.mkdir(parents=True, exist_ok=True)


def extract_code_strings(path: Path) -> List[str]:
    data = path.read_bytes()
    pattern = re.compile(r"(\d+\.[\u4e00-\u9fa5A-Za-z（）()·、\s]+)")
    seen: List[str] = []
    for offset in (0, 1):
        text = data[offset:].decode("utf-16le", errors="ignore")
        for match in pattern.findall(text):
            item = match.strip()
            if item and item not in seen:
                seen.append(item)
    return seen


def collect_personal_identity(strings: Sequence[str]) -> List[str]:
    result: List[str] = []
    for item in strings:
        if any(keyword in item for keyword in KEYWORDS_PERSONAL_ID):
            if item not in result:
                result.append(item)
    return result


def collect_household_types(strings: Sequence[str]) -> List[str]:
    result: List[str] = []
    for item in strings:
        if "城镇" in item or "农村" in item:
            if item not in result:
                result.append(item)
    return result


def collect_education_levels(strings: Sequence[str]) -> List[str]:
    result: List[str] = []
    for item in strings:
        if any(keyword in item for keyword in KEYWORDS_EDUCATION):
            if item not in result:
                result.append(item)
    return result


def collect_special_skills(strings: Sequence[str]) -> List[str]:
    result: List[str] = []
    for item in strings:
        code, _, _ = item.partition(".")
        if code in SPECIAL_SKILL_CODES and item not in result:
            result.append(item)
    return result


def collect_skill_levels(strings: Sequence[str]) -> List[str]:
    result: List[str] = []
    for item in strings:
        code, _, _ = item.partition(" ")
        code_alt, _, _ = item.partition(".")
        if code in SKILL_LEVEL_CODES or code_alt in SKILL_LEVEL_CODES:
            if item not in result:
                result.append(item)
    if "无" not in result:
        result.append("无")
    return result


def collect_reduction_flags(path: Path) -> List[str]:
    data = path.read_bytes()
    pattern = re.compile(r"(\d+\s*[\u4e00-\u9fa5（）()·、]+)")
    result: List[str] = []
    for offset in (0, 1):
        text = data[offset:].decode("utf-16le", errors="ignore")
        for match in pattern.findall(text):
            item = match.strip()
            if "减少" in item and item not in result:
                result.append(item)
    return result


def extract_default_values() -> dict:
    book = xlrd.open_workbook(str(INCREASE_TEMPLATE))
    sheet = book.sheet_by_name("批量导入")
    sample = sheet.row_values(3)
    return {
        "personal_identity": str(sample[7]).strip(),
        "household_type": str(sample[8]).strip(),
        "education_level": str(sample[10]).strip(),
        "special_skill": str(sample[17]).strip(),
        "skill_level": str(sample[18]).strip(),
    }


def extract_decrease_defaults() -> dict:
    book = xlrd.open_workbook(str(DECREASE_TEMPLATE))
    sheet = book.sheet_by_name("批量导入")
    sample = sheet.row_values(3)
    return {
        "decrease_reason": str(sample[5]).strip(),
        "unemployment_reason": str(sample[11]).strip(),
        "reduction_flag": str(sample[6]).strip(),
    }


def collect_decrease_reasons() -> List[str]:
    book = xlrd.open_workbook(str(DECREASE_TEMPLATE))
    sheet = book.sheet_by_name("二级代码")
    reasons: List[str] = []
    for col in range(sheet.ncols):
        value = str(sheet.cell_value(0, col)).strip()
        if value:
            reasons.append(value)
    return reasons


def collect_unemployment_reasons() -> List[str]:
    book = xlrd.open_workbook(str(DECREASE_TEMPLATE))
    sheet = book.sheet_by_name("二级代码")
    reasons: List[str] = []
    for row in range(1, 4):
        for col in range(sheet.ncols):
            value = str(sheet.cell_value(row, col)).strip()
            if value:
                reasons.append(value)
    return reasons


def build_option_set(options: Iterable[str], default_value: str) -> OptionSet:
    items = [item for item in options if item]
    unique: List[str] = []
    for item in items:
        if item not in unique:
            unique.append(item)
    default = default_value if default_value in unique else (unique[0] if unique else "")
    return OptionSet(unique, default)


def main() -> None:
    ensure_output_dir()
    code_strings = extract_code_strings(INCREASE_TEMPLATE)
    defaults = extract_default_values()
    decrease_defaults = extract_decrease_defaults()

    personal_identity = collect_personal_identity(code_strings)
    household_types = collect_household_types(code_strings)
    education_levels = collect_education_levels(code_strings)
    special_skills = collect_special_skills(code_strings)
    skill_levels = collect_skill_levels(code_strings)

    decrease_reasons = collect_decrease_reasons()
    unemployment_reasons = collect_unemployment_reasons()
    reduction_flags = collect_reduction_flags(DECREASE_TEMPLATE)

    payload = {
        "generated_at": datetime.utcnow().isoformat(timespec="seconds") + "Z",
        "personal_identity": build_option_set(personal_identity, defaults["personal_identity"]).as_dict(),
        "household_type": build_option_set(household_types, defaults["household_type"]).as_dict(),
        "education_level": build_option_set(education_levels, defaults["education_level"]).as_dict(),
        "special_skill": build_option_set(special_skills, defaults["special_skill"]).as_dict(),
        "skill_level": build_option_set(skill_levels, defaults["skill_level"]).as_dict(),
        "decrease_reason": build_option_set(decrease_reasons, decrease_defaults["decrease_reason"]).as_dict(),
        "unemployment_reason": build_option_set(unemployment_reasons, decrease_defaults["unemployment_reason"]).as_dict(),
        "reduction_flag": build_option_set(reduction_flags, decrease_defaults["reduction_flag"]).as_dict(),
    }

    OUTPUT_JSON.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"Generated {OUTPUT_JSON.relative_to(ROOT)} with {len(code_strings)} template entries")


if __name__ == "__main__":
    main()

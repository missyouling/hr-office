"use client";

import { useState, useRef, useCallback, useEffect, useMemo } from "react";
import { toast } from "sonner";
import {
  Plus,
  Upload,
  Download,
  Search,
  MoreVertical,
  UserMinus,
  CalendarPlus,
  CalendarMinus,
  Settings,
  Dice6
} from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { fetchEmployees, importEmployees as importEmployeesApi, EmployeeResponse } from "@/lib/api";
import { useAuth } from "@/lib/auth";

// 定义所有可用的字段
interface FieldOption {
  key: keyof Employee;
  label: string;
  width?: string;
}

const AVAILABLE_FIELDS: FieldOption[] = [
  { key: "employeeId", label: "工号", width: "80px" },
  { key: "name", label: "姓名", width: "100px" },
  { key: "department", label: "部门", width: "120px" },
  { key: "position", label: "岗位", width: "120px" },
  { key: "gender", label: "性别", width: "60px" },
  { key: "hireDate", label: "入职时间", width: "120px" },
  { key: "age", label: "年龄", width: "60px" },
  { key: "workYears", label: "工龄", width: "60px" },
  { key: "birthMonth", label: "出生月份", width: "80px" },
  { key: "education", label: "文化程度", width: "100px" },
  { key: "politicalStatus", label: "政治面貌", width: "100px" },
  { key: "workClothingSize", label: "工作服", width: "80px" },
  { key: "safetyShoeSize", label: "劳保鞋", width: "80px" },
  { key: "householdType", label: "户口性质", width: "80px" },
  { key: "ethnicity", label: "民族", width: "80px" },
  { key: "nativePlace", label: "籍贯", width: "100px" },
  { key: "idAddress", label: "身份证地址", width: "250px" },
  { key: "idNumber", label: "身份证号码", width: "200px" },
  { key: "maritalStatus", label: "婚姻状况", width: "80px" },
  { key: "socialInsurance", label: "社保", width: "60px" },
  { key: "hasBirth", label: "是否生育", width: "80px" },
  { key: "phone", label: "联系电话", width: "120px" },
  { key: "emergencyContact", label: "紧急联系人", width: "100px" },
  { key: "emergencyPhone", label: "紧急联系电话", width: "200px" },
  { key: "currentAddress", label: "现居住地址", width: "250px" },
  { key: "graduateSchool", label: "毕业院校", width: "150px" },
  { key: "major", label: "专业", width: "120px" },
  { key: "graduationTime", label: "毕业时间", width: "100px" },
];

// 默认显示的字段
const DEFAULT_VISIBLE_FIELDS: (keyof Employee)[] = [
  "employeeId", "name", "department", "position", "gender", "hireDate", "phone"
];

interface SearchableSelectProps {
  id: string;
  value: string;
  onChange: (value: string) => void;
  options: string[];
  placeholder: string;
}

const SearchableSelect = ({ id, value, onChange, options, placeholder }: SearchableSelectProps) => {
  const sanitizedOptions = options
    .map((option) => option.trim())
    .filter((option) => option.length > 0);

  if (sanitizedOptions.length === 0) {
    return (
      <Input
        id={id}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
      />
    );
  }

  if (sanitizedOptions.length > 3) {
    const dataListId = `${id}-options`;
    return (
      <>
        <Input
          id={id}
          list={dataListId}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder={placeholder}
        />
        <datalist id={dataListId}>
          {sanitizedOptions.map((option) => (
            <option key={option} value={option} />
          ))}
        </datalist>
      </>
    );
  }

  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger id={id}>
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        {sanitizedOptions.map((option) => (
          <SelectItem key={option} value={option}>
            {option}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};

interface Employee {
  id: string;
  // 基本信息 - 对应Excel模板字段
  employeeId?: string;        // 工号
  name: string;               // 姓名
  department: string;         // 部门
  position: string;           // 岗位
  gender?: string;            // 性别
  hireDate: string;           // 入职时间
  age?: string;               // 年龄
  workYears?: string;         // 工龄
  birthMonth?: string;        // 出生月份
  education?: string;         // 文化程度
  politicalStatus?: string;   // 政治面貌
  workClothingSize?: string;  // 工作服
  safetyShoeSize?: string;    // 劳保鞋
  householdType?: string;     // 户口性质
  ethnicity?: string;         // 民族
  nativePlace?: string;       // 籍贯
  idAddress?: string;         // 身份证地址
  idNumber: string;           // 身份证号码
  maritalStatus?: string;     // 婚姻状况
  socialInsurance?: string;   // 社保
  hasBirth?: string;          // 是否生育
  phone?: string;             // 联系电话
  emergencyContact?: string;  // 紧急联系人
  emergencyPhone?: string;    // 家庭电话/紧急情况联系电话
  currentAddress?: string;    // 现居住地址
  graduateSchool?: string;    // 毕业院校
  major?: string;             // 专业
  graduationTime?: string;    // 毕业时间

  // 系统字段
  email?: string;
  remarks?: string;
  status: 'active' | 'resigned';
  resignDate?: string;
}

const normalizeApiString = (value?: string | null) => (value ?? "").trim();

const generateClientId = () => {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
};

const MS_PER_DAY = 1000 * 60 * 60 * 24;
const MS_PER_YEAR = MS_PER_DAY * 365.25;

const formatDateToInput = (date: Date) => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
};

const normalizeDateInput = (value?: string | null) => {
  if (!value) return "";
  let cleaned = value.trim();
  if (!cleaned) return "";
  cleaned = cleaned.replace(/[.\/]/g, "-");
  if (/^\d{8}$/.test(cleaned)) {
    return `${cleaned.slice(0, 4)}-${cleaned.slice(4, 6)}-${cleaned.slice(6, 8)}`;
  }
  const match = cleaned.match(/^(\d{4})-(\d{1,2})-(\d{1,2})/);
  if (match) {
    const [, y, m, d] = match;
    return `${y}-${m.padStart(2, "0")}-${d.padStart(2, "0")}`;
  }
  return cleaned.slice(0, 10);
};

const formatAgeValue = (value?: string | null) => {
  if (!value) return "";
  const num = parseInt(value, 10);
  if (Number.isNaN(num) || num <= 0) {
    return "";
  }
  return String(num);
};

const formatWorkYearsValue = (value?: string | null) => {
  if (!value) return "";
  const num = Number(value);
  if (Number.isNaN(num) || num < 0) {
    return "";
  }
  return num.toFixed(2);
};

const normalizeBirthMonth = (value?: string | null) => {
  if (!value) return "";
  const trimmed = value.trim();
  if (!trimmed) return "";
  if (/^\d{1,2}月$/.test(trimmed)) {
    const month = trimmed.replace("月", "");
    return `${month.padStart(2, "0")}月`;
  }
  if (/^\d{4}-\d{1,2}$/.test(trimmed)) {
    const month = trimmed.split("-")[1];
    return `${month.padStart(2, "0")}月`;
  }
  if (/^\d{6}$/.test(trimmed)) {
    return `${trimmed.slice(4, 6)}月`;
  }
  return trimmed;
};

const formatFieldDisplayValue = (employee: Employee, fieldKey: keyof Employee) => {
  const raw = employee[fieldKey];
  if (!raw) {
    return "-";
  }
  if (fieldKey === "hireDate") {
    return normalizeDateInput(raw as string) || "-";
  }
  if (fieldKey === "age") {
    return formatAgeValue(raw as string) || "-";
  }
  if (fieldKey === "workYears") {
    return formatWorkYearsValue(raw as string) || "-";
  }
  return raw;
};

const mapEmployeeFromApi = (employee: EmployeeResponse): Employee => ({
  id: employee.id ? String(employee.id) : generateClientId(),
  employeeId: normalizeApiString(employee.employee_id),
  name: normalizeApiString(employee.name),
  department: normalizeApiString(employee.department),
  position: normalizeApiString(employee.position),
  gender: normalizeApiString(employee.gender),
  hireDate: normalizeDateInput(employee.hire_date),
  age: formatAgeValue(employee.age),
  workYears: formatWorkYearsValue(employee.work_years),
  birthMonth: normalizeBirthMonth(employee.birth_month),
  education: normalizeApiString(employee.education),
  politicalStatus: normalizeApiString(employee.political_status),
  workClothingSize: normalizeApiString(employee.work_clothing_size),
  safetyShoeSize: normalizeApiString(employee.safety_shoe_size),
  householdType: normalizeApiString(employee.household_type),
  ethnicity: normalizeApiString(employee.ethnicity),
  nativePlace: normalizeApiString(employee.native_place),
  idAddress: normalizeApiString(employee.id_address),
  idNumber: normalizeApiString(employee.id_number),
  maritalStatus: normalizeApiString(employee.marital_status),
  socialInsurance: normalizeApiString(employee.social_insurance),
  hasBirth: normalizeApiString(employee.has_birth),
  phone: normalizeApiString(employee.phone),
  emergencyContact: normalizeApiString(employee.emergency_contact),
  emergencyPhone: normalizeApiString(employee.emergency_phone),
  currentAddress: normalizeApiString(employee.current_address),
  graduateSchool: normalizeApiString(employee.graduate_school),
  major: normalizeApiString(employee.major),
  graduationTime: normalizeApiString(employee.graduation_time),
  email: normalizeApiString(employee.email),
  remarks: normalizeApiString(employee.remarks),
  status: employee.status === "resigned" ? "resigned" : "active",
  resignDate: normalizeDateInput(employee.resign_date),
});

const createEmptyEmployee = (): Partial<Employee> => ({
  employeeId: "",
  name: "",
  department: "",
  position: "",
  gender: "",
  hireDate: formatDateToInput(new Date()),
  age: "",
  workYears: "",
  birthMonth: "",
  education: "",
  politicalStatus: "",
  workClothingSize: "",
  safetyShoeSize: "",
  householdType: "",
  ethnicity: "",
  nativePlace: "",
  idAddress: "",
  idNumber: "",
  maritalStatus: "",
  socialInsurance: "",
  hasBirth: "",
  phone: "",
  emergencyContact: "",
  emergencyPhone: "",
  currentAddress: "",
  graduateSchool: "",
  major: "",
  graduationTime: "",
  email: "",
  remarks: "",
});

interface SocialInsuranceChange {
  id: string;
  employeeId: string;
  employeeName: string;
  department: string;
  type: 'increase' | 'decrease';
  effectiveDate: string;
  reason?: string;
  createDate: string;
}

interface EmployeeManagementProps {
  className?: string;
}

export function EmployeeManagement({ className }: EmployeeManagementProps) {
  // 状态管理
  const [employees, setEmployees] = useState<Employee[]>([]);
  const [resignedEmployees, setResignedEmployees] = useState<Employee[]>([]);
  const [socialInsuranceChanges, setSocialInsuranceChanges] = useState<SocialInsuranceChange[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const { token, isAuthenticated, isLoading: authLoading } = useAuth();
  const applyEmployeeData = useCallback((list: EmployeeResponse[]) => {
    const mapped = list.map(mapEmployeeFromApi);
    const active = mapped.filter((emp) => emp.status !== "resigned");
    const resigned = mapped.filter((emp) => emp.status === "resigned");
    setEmployees(active);
    setResignedEmployees(resigned);
  }, [setEmployees, setResignedEmployees]);

  const loadEmployees = useCallback(async () => {
    if (!token) {
      return;
    }
    try {
      const data = await fetchEmployees(token);
      applyEmployeeData(data);
    } catch (error) {
      console.error("[EmployeeManagement] failed to load employees", error);
      toast.error(
        error instanceof Error ? error.message : "获取员工列表失败，请稍后重试",
      );
    }
  }, [applyEmployeeData, token]);

  useEffect(() => {
    if (!authLoading && isAuthenticated) {
      loadEmployees();
    }
  }, [authLoading, isAuthenticated, loadEmployees]);

  // 对话框状态
  const [showAddEmployee, setShowAddEmployee] = useState(false);
  const [showBatchImport, setShowBatchImport] = useState(false);
  const [showResignDialog, setShowResignDialog] = useState(false);
  const [showSocialInsuranceDialog, setShowSocialInsuranceDialog] = useState(false);
  const [showSingleSocialInsuranceDialog, setShowSingleSocialInsuranceDialog] = useState(false);
  const [showEditEmployee, setShowEditEmployee] = useState(false);

  // 当前操作的员工
  const [selectedEmployee, setSelectedEmployee] = useState<Employee | null>(null);

  // 编辑员工表单数据
  const [editEmployee, setEditEmployee] = useState<Partial<Employee>>({});

  // 部门编码映射
  const DEPARTMENT_CODES: Record<string, string> = {
    "总经办": "C020100",
    "财务部": "C020200",
    "销售部": "C020300",
    "仓库": "C020400",
    "生产部": "C020500",
    "机电部": "C020510",
    "技术质量部": "C020600",
    "人事行政部": "C020700",
  };

  // 表单数据（包含默认值）
  const [newEmployee, setNewEmployee] = useState<Partial<Employee>>(() => ({
    ...createEmptyEmployee(),
    education: "初中",
    politicalStatus: "群众",
    workClothingSize: "L",
    safetyShoeSize: "40",
    socialInsurance: "有",
  }));

  const [resignDate, setResignDate] = useState("");
  const [socialInsuranceChange, setSocialInsuranceChange] = useState({
    type: "increase" as "increase" | "decrease",
    effectiveDate: "",
    reason: "",
    selectedEmployees: [] as string[],
  });
  const [singleSocialInsuranceChange, setSingleSocialInsuranceChange] = useState({
    type: "increase" as "increase" | "decrease",
    effectiveDate: "",
    reason: "",
  });

  // 搜索和筛选
  const [searchTerm, setSearchTerm] = useState("");
  const [departmentFilter, setDepartmentFilter] = useState("all");

  // 字段显示控制
  const [visibleFields, setVisibleFields] = useState<(keyof Employee)[]>(() => {
    // 从localStorage读取用户偏好，如果没有则使用默认值
    if (typeof window !== 'undefined') {
      const saved = localStorage.getItem('employee-visible-fields');
      return saved ? JSON.parse(saved) : DEFAULT_VISIBLE_FIELDS;
    }
    return DEFAULT_VISIBLE_FIELDS;
  });
  const [showFieldSelector, setShowFieldSelector] = useState(false);

  // 文件上传引用
  const fileInputRef = useRef<HTMLInputElement>(null);

  // 获取所有部门列表
  const getDepartments = () => {
    const depts = new Set<string>();
    Object.keys(DEPARTMENT_CODES).forEach((dept) => depts.add(dept));
    employees.forEach(emp => {
      const dept = emp.department?.trim();
      if (dept) depts.add(dept);
    });
    resignedEmployees.forEach(emp => {
      const dept = emp.department?.trim();
      if (dept) depts.add(dept);
    });
    return Array.from(depts).sort((a, b) => a.localeCompare(b, "zh-CN"));
  };

  const positionOptions = useMemo(() => {
    const set = new Set<string>();
    [...employees, ...resignedEmployees].forEach((emp) => {
      const value = emp.position?.trim();
      if (value) set.add(value);
    });
    return Array.from(set).sort((a, b) => a.localeCompare(b, "zh-CN"));
  }, [employees, resignedEmployees]);

  const nativePlaceOptions = useMemo(() => {
    const set = new Set<string>();
    [...employees, ...resignedEmployees].forEach((emp) => {
      const value = emp.nativePlace?.trim();
      if (value) set.add(value);
    });
    return Array.from(set).sort((a, b) => a.localeCompare(b, "zh-CN"));
  }, [employees, resignedEmployees]);

  const educationOptions = useMemo(() => {
    const defaults = ["小学", "初中", "高中", "中专", "大专", "本科", "硕士", "博士"];
    const set = new Set<string>(defaults);
    [...employees, ...resignedEmployees].forEach((emp) => {
      const value = emp.education?.trim();
      if (value) set.add(value);
    });
    return Array.from(set).sort((a, b) => a.localeCompare(b, "zh-CN"));
  }, [employees, resignedEmployees]);

  const graduateSchoolOptions = useMemo(() => {
    const set = new Set<string>();
    [...employees, ...resignedEmployees].forEach((emp) => {
      const value = emp.graduateSchool?.trim();
      if (value) set.add(value);
    });
    return Array.from(set).sort((a, b) => a.localeCompare(b, "zh-CN"));
  }, [employees, resignedEmployees]);

  const majorOptions = useMemo(() => {
    const set = new Set<string>();
    [...employees, ...resignedEmployees].forEach((emp) => {
      const value = emp.major?.trim();
      if (value) set.add(value);
    });
    return Array.from(set).sort((a, b) => a.localeCompare(b, "zh-CN"));
  }, [employees, resignedEmployees]);

  // 字段管理函数
  const handleFieldToggle = (fieldKey: keyof Employee) => {
    const newVisibleFields = visibleFields.includes(fieldKey)
      ? visibleFields.filter(key => key !== fieldKey)
      : [...visibleFields, fieldKey];

    setVisibleFields(newVisibleFields);

    // 保存到localStorage
    if (typeof window !== 'undefined') {
      localStorage.setItem('employee-visible-fields', JSON.stringify(newVisibleFields));
    }
  };

  const resetFieldsToDefault = () => {
    setVisibleFields(DEFAULT_VISIBLE_FIELDS);
    if (typeof window !== 'undefined') {
      localStorage.setItem('employee-visible-fields', JSON.stringify(DEFAULT_VISIBLE_FIELDS));
    }
  };

  // 获取当前显示的字段配置
  const getVisibleFieldsConfig = () => {
    return AVAILABLE_FIELDS.filter(field => visibleFields.includes(field.key));
  };

  // 筛选员工
  const filteredEmployees = employees.filter(emp => {
    const matchesSearch = !searchTerm ||
      emp.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      emp.idNumber.includes(searchTerm) ||
      emp.department?.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesDepartment = !departmentFilter || departmentFilter === "all" || emp.department === departmentFilter;

    return matchesSearch && matchesDepartment;
  });

  // 筛选离职员工
  const filteredResignedEmployees = resignedEmployees.filter(emp => {
    const matchesSearch = !searchTerm ||
      emp.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      emp.idNumber.includes(searchTerm) ||
      emp.department?.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesDepartment = !departmentFilter || departmentFilter === "all" || emp.department === departmentFilter;

    return matchesSearch && matchesDepartment;
  });

  // 添加员工
  const handleAddEmployee = () => {
    // 必填字段验证
    if (!newEmployee.name?.trim()) {
      toast.error("请填写姓名");
      return;
    }
    if (!newEmployee.idNumber?.trim()) {
      toast.error("请填写身份证号");
      return;
    }
    if (!newEmployee.department?.trim()) {
      toast.error("请选择部门");
      return;
    }

    // 身份证号格式验证
    const idRegex = /^[1-9]\d{5}(18|19|([23]\d))\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]$/;
    if (!idRegex.test(newEmployee.idNumber)) {
      toast.error("身份证号格式不正确");
      return;
    }

    // 手机号格式验证（如果填写了）
    if (newEmployee.phone && !/^1[3-9]\d{9}$/.test(newEmployee.phone)) {
      toast.error("手机号格式不正确");
      return;
    }

    // 邮箱格式验证（如果填写了）
    if (newEmployee.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(newEmployee.email)) {
      toast.error("邮箱格式不正确");
      return;
    }

    // 年龄验证（如果填写了）
    if (newEmployee.age && (isNaN(Number(newEmployee.age)) || Number(newEmployee.age) < 16 || Number(newEmployee.age) > 100)) {
      toast.error("年龄应为16-100之间的数字");
      return;
    }

    // 工龄验证（如果填写了）
    if (newEmployee.workYears && (isNaN(Number(newEmployee.workYears)) || Number(newEmployee.workYears) < 0)) {
      toast.error("工龄应为非负数字");
      return;
    }

    // 检查身份证号是否重复
    const sanitizedIdNumber = newEmployee.idNumber.trim().toUpperCase();
    const existingEmployee = employees.find(emp => emp.idNumber === sanitizedIdNumber);
    if (existingEmployee) {
      toast.error("该身份证号已存在");
      return;
    }

    const autoInfo = calculateAgeFromIdNumber(sanitizedIdNumber);
    const normalizedHireDate = normalizeDateInput(newEmployee.hireDate || formatDateToInput(new Date()));
    const normalizedAge = formatAgeValue(newEmployee.age || autoInfo.age);
    const normalizedWorkYears = formatWorkYearsValue(newEmployee.workYears || calculateWorkYears(normalizedHireDate));
    const normalizedBirthMonth = normalizeBirthMonth(newEmployee.birthMonth || autoInfo.birthMonth);

    const employee: Employee = {
      id: Date.now().toString(),
      employeeId: newEmployee.employeeId || Date.now().toString(),
      name: newEmployee.name.trim(),
      department: newEmployee.department.trim(),
      position: newEmployee.position?.trim() || "",
      gender: newEmployee.gender || "",
      hireDate: normalizedHireDate,
      age: normalizedAge,
      workYears: normalizedWorkYears,
      birthMonth: normalizedBirthMonth,
      education: newEmployee.education?.trim() || "",
      politicalStatus: newEmployee.politicalStatus?.trim() || "",
      workClothingSize: newEmployee.workClothingSize?.trim() || "",
      safetyShoeSize: newEmployee.safetyShoeSize?.trim() || "",
      householdType: newEmployee.householdType?.trim() || "",
      ethnicity: newEmployee.ethnicity?.trim() || "",
      nativePlace: newEmployee.nativePlace?.trim() || "",
      idAddress: newEmployee.idAddress?.trim() || "",
      idNumber: sanitizedIdNumber,
      maritalStatus: newEmployee.maritalStatus?.trim() || "",
      socialInsurance: newEmployee.socialInsurance?.trim() || "",
      hasBirth: newEmployee.hasBirth?.trim() || "",
      phone: newEmployee.phone?.trim() || "",
      emergencyContact: newEmployee.emergencyContact?.trim() || "",
      emergencyPhone: newEmployee.emergencyPhone?.trim() || "",
      currentAddress: newEmployee.currentAddress?.trim() || "",
      graduateSchool: newEmployee.graduateSchool?.trim() || "",
      major: newEmployee.major?.trim() || "",
      graduationTime: normalizeDateInput(newEmployee.graduationTime),
      email: newEmployee.email?.trim() || "",
      remarks: newEmployee.remarks?.trim() || "",
      status: 'active',
    };

    setEmployees(prev => [...prev, employee]);
    setNewEmployee({
      ...createEmptyEmployee(),
      education: "初中",
      politicalStatus: "群众",
      workClothingSize: "L",
      safetyShoeSize: "40",
      socialInsurance: "有",
    });
    setShowAddEmployee(false);
    toast.success("员工添加成功");
  };

  // 批量导入员工
  const handleBatchImport = async () => {
    const file = fileInputRef.current?.files?.[0];
    if (!file) {
      toast.error("请选择文件");
      return;
    }

    setIsLoading(true);
    try {
    const token = localStorage.getItem("token");
    if (!token) {
      toast.error("登录状态失效，请重新登录后再导入");
      setIsLoading(false);
      return;
    }

      if (!token) {
        toast.error("登录状态失效，请重新登录后再导入");
        return;
      }

      const result = await importEmployeesApi(file, token);
      applyEmployeeData(result.employees);
      setShowBatchImport(false);
      const skippedMsg = result.skipped
        ? `，已跳过 ${result.skipped} 行无效数据`
        : "";
      toast.success(`批量导入成功，共导入 ${result.imported} 人${skippedMsg}`);
    } catch (error) {
      console.error("[EmployeeImport] failed", error);
      toast.error(
        error instanceof Error
          ? error.message
          : "导入失败，请确认文件符合模板要求",
      );
    } finally {
      setIsLoading(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    }
  };

  // 员工离职
  const handleResignEmployee = () => {
    if (!selectedEmployee || !resignDate) {
      toast.error("请选择离职日期");
      return;
    }

    const normalizedResignDate = normalizeDateInput(resignDate);

    const resignedEmployee: Employee = {
      ...selectedEmployee,
      status: 'resigned',
      resignDate: normalizedResignDate,
    };

    setEmployees(prev => prev.filter(emp => emp.id !== selectedEmployee.id));
    setResignedEmployees(prev => [...prev, resignedEmployee]);

    setShowResignDialog(false);
    setSelectedEmployee(null);
    setResignDate("");
    toast.success("员工离职处理成功");
  };

  // 添加社保增减记录
  const handleSocialInsuranceChange = () => {
    if (socialInsuranceChange.selectedEmployees.length === 0 || !socialInsuranceChange.effectiveDate) {
      toast.error("请选择员工和生效日期");
      return;
    }

    const changes = socialInsuranceChange.selectedEmployees.map(empId => {
      const employee = employees.find(emp => emp.id === empId);
      return {
        id: Date.now().toString() + empId,
        employeeId: empId,
        employeeName: employee?.name || "",
        department: employee?.department?.trim() || "",
        type: socialInsuranceChange.type,
        effectiveDate: socialInsuranceChange.effectiveDate,
        reason: socialInsuranceChange.reason,
        createDate: new Date().toISOString().split('T')[0],
      };
    });

    setSocialInsuranceChanges(prev => [...prev, ...changes]);
    setSocialInsuranceChange({
      type: "increase",
      effectiveDate: "",
      reason: "",
      selectedEmployees: [],
    });
    setShowSocialInsuranceDialog(false);
    toast.success("社保增减记录添加成功");
  };

  // 处理单个员工社保增减
  const handleSingleSocialInsuranceChange = () => {
    if (!selectedEmployee || !singleSocialInsuranceChange.effectiveDate) {
      toast.error("请选择生效日期");
      return;
    }

    const change: SocialInsuranceChange = {
      id: Date.now().toString(),
      employeeId: selectedEmployee.id,
      employeeName: selectedEmployee.name,
      department: selectedEmployee.department.trim(),
      type: singleSocialInsuranceChange.type,
      effectiveDate: singleSocialInsuranceChange.effectiveDate,
      reason: singleSocialInsuranceChange.reason,
      createDate: new Date().toISOString().split('T')[0],
    };

    setSocialInsuranceChanges(prev => [...prev, change]);
    setShowSingleSocialInsuranceDialog(false);
    setSelectedEmployee(null);
    setSingleSocialInsuranceChange({
      type: "increase",
      effectiveDate: "",
      reason: "",
    });

    const typeText = singleSocialInsuranceChange.type === "increase" ? "增加" : "减少";
    toast.success(`${selectedEmployee.name} 社保${typeText}记录添加成功`);
  };

  const handleRestoreEmployee = (employee: Employee) => {
    setResignedEmployees((prev) => prev.filter((item) => item.id !== employee.id));
    setEmployees((prev) => [...prev, { ...employee, status: "active", resignDate: "", department: employee.department.trim() }]);
    toast.success(`已恢复 ${employee.name} 为在职状态`);
  };

  const handleRemoveResignedEmployee = (employee: Employee) => {
    setResignedEmployees((prev) => prev.filter((item) => item.id !== employee.id));
    toast.success(`已删除离职员工 ${employee.name} 的记录`);
  };

  const handleRemoveSocialInsuranceChange = (id: string) => {
    setSocialInsuranceChanges((prev) => prev.filter((change) => change.id !== id));
    toast.success("已撤销该社保变动记录");
  };

  // 下载模板
  const downloadTemplate = () => {
    // 这里应该调用API下载模板
    toast.info("正在下载模板...");
  };

  // 从身份证号码计算年龄和出生月份
  const calculateAgeFromIdNumber = (idNumber: string) => {
    const normalized = idNumber.replace(/\s+/g, "").toUpperCase();
    if (!normalized || normalized.length !== 18) return { age: "", birthMonth: "" };

    const birth = normalized.substring(6, 14);
    const birthYear = parseInt(birth.substring(0, 4));
    const birthMonth = parseInt(birth.substring(4, 6));
    const birthDay = parseInt(birth.substring(6, 8));

    const today = new Date();
    const currentYear = today.getFullYear();
    const currentMonth = today.getMonth() + 1;
    const currentDay = today.getDate();

    let age = currentYear - birthYear;
    if (currentMonth < birthMonth || (currentMonth === birthMonth && currentDay < birthDay)) {
      age--;
    }

    return {
      age: age.toString(),
      birthMonth: `${String(birthMonth).padStart(2, "0")}月`
    };
  };

  // 从入职时间计算工龄
  const calculateWorkYears = (hireDate: string) => {
    const normalized = normalizeDateInput(hireDate);
    if (!normalized) return "";

    const hire = new Date(normalized);
    const today = new Date();
    const diffTime = Math.abs(today.getTime() - hire.getTime());
    if (!Number.isFinite(diffTime)) return "";
    const diffYears = diffTime / MS_PER_YEAR;
    return formatWorkYearsValue(diffYears.toString());
  };

  // 生成工号
  const generateEmployeeId = (department: string) => {
    const departmentCode = DEPARTMENT_CODES[department as keyof typeof DEPARTMENT_CODES];
    if (!departmentCode) return "";

    // 查找该部门最大的工号
    const departmentEmployees = [...employees, ...resignedEmployees]
      .filter(emp => emp.employeeId && emp.employeeId.startsWith(departmentCode))
      .map(emp => emp.employeeId!)
      .sort();

    if (departmentEmployees.length === 0) {
      return `${departmentCode}001`;
    }

    const lastId = departmentEmployees[departmentEmployees.length - 1];
    if (!lastId) {
      return `${departmentCode}001`;
    }
    
    const lastNumber = parseInt(lastId.substring(departmentCode.length));
    const nextNumber = lastNumber + 1;

    return `${departmentCode}${nextNumber.toString().padStart(3, '0')}`;
  };

  // 自动生成工号
  const handleGenerateEmployeeId = () => {
    if (!newEmployee.department) {
      toast.error("请先选择部门");
      return;
    }

    const generatedId = generateEmployeeId(newEmployee.department);
    setNewEmployee(prev => ({ ...prev, employeeId: generatedId }));
    toast.success(`已生成工号：${generatedId}`);
  };

  // 处理身份证号变化
  const handleIdNumberChange = (idNumber: string, isEdit = false) => {
    const normalized = idNumber.replace(/\s+/g, "").toUpperCase();
    const { age, birthMonth } = calculateAgeFromIdNumber(normalized);

    if (isEdit) {
      setEditEmployee(prev => ({
        ...prev,
        idNumber: normalized,
        age: age || prev.age,
        birthMonth: birthMonth || prev.birthMonth
      }));
    } else {
      setNewEmployee(prev => ({
        ...prev,
        idNumber: normalized,
        age: age || prev.age,
        birthMonth: birthMonth || prev.birthMonth
      }));
    }
  };

  // 处理入职时间变化
  const handleHireDateChange = (hireDate: string, isEdit = false) => {
    const normalized = normalizeDateInput(hireDate);
    const workYears = calculateWorkYears(normalized);

    if (isEdit) {
      setEditEmployee(prev => ({
        ...prev,
        hireDate: normalized,
        workYears: workYears || prev.workYears
      }));
    } else {
      setNewEmployee(prev => ({
        ...prev,
        hireDate: normalized,
        workYears: workYears || prev.workYears
      }));
    }
  };

  const handleAgeBlur = (value: string, isEdit = false) => {
    const normalized = formatAgeValue(value);
    if (isEdit) {
      setEditEmployee(prev => ({ ...prev, age: normalized }));
    } else {
      setNewEmployee(prev => ({ ...prev, age: normalized }));
    }
  };

  const handleWorkYearsBlur = (value: string, isEdit = false) => {
    const normalized = formatWorkYearsValue(value);
    if (isEdit) {
      setEditEmployee(prev => ({ ...prev, workYears: normalized }));
    } else {
      setNewEmployee(prev => ({ ...prev, workYears: normalized }));
    }
  };

  // 编辑员工
  const handleEditEmployee = (employee: Employee) => {
    setEditEmployee({ ...employee });
    setShowEditEmployee(true);
  };

  // 保存编辑的员工信息
  const handleSaveEmployee = () => {
    // 必填字段验证
    if (!editEmployee.name?.trim()) {
      toast.error("请填写姓名");
      return;
    }
    if (!editEmployee.idNumber?.trim()) {
      toast.error("请填写身份证号");
      return;
    }
    if (!editEmployee.department?.trim()) {
      toast.error("请选择部门");
      return;
    }

    // 身份证号格式验证
    const idRegex = /^[1-9]\d{5}(18|19|([23]\d))\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]$/;
    if (!idRegex.test(editEmployee.idNumber)) {
      toast.error("身份证号格式不正确");
      return;
    }

    // 手机号格式验证（如果填写了）
    if (editEmployee.phone && !/^1[3-9]\d{9}$/.test(editEmployee.phone)) {
      toast.error("手机号格式不正确");
      return;
    }

    // 邮箱格式验证（如果填写了）
    if (editEmployee.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(editEmployee.email)) {
      toast.error("邮箱格式不正确");
      return;
    }

    // 年龄验证（如果填写了）
    if (editEmployee.age && (isNaN(Number(editEmployee.age)) || Number(editEmployee.age) < 16 || Number(editEmployee.age) > 100)) {
      toast.error("年龄应为16-100之间的数字");
      return;
    }

    // 工龄验证（如果填写了）
    if (editEmployee.workYears && (isNaN(Number(editEmployee.workYears)) || Number(editEmployee.workYears) < 0)) {
      toast.error("工龄应为非负数字");
      return;
    }

    // 检查身份证号是否重复（排除当前员工）
    const sanitizedIdNumber = editEmployee.idNumber.trim().toUpperCase();
    const existingEmployee = [...employees, ...resignedEmployees].find(
      emp => emp.idNumber === sanitizedIdNumber && emp.id !== editEmployee.id
    );
    if (existingEmployee) {
      toast.error("该身份证号已存在");
      return;
    }

    const autoInfo = calculateAgeFromIdNumber(sanitizedIdNumber);
    const normalizedHireDate = normalizeDateInput(editEmployee.hireDate || "");
    const normalizedAge = formatAgeValue(editEmployee.age || autoInfo.age);
    const normalizedWorkYears = formatWorkYearsValue(editEmployee.workYears || calculateWorkYears(normalizedHireDate));
    const normalizedBirthMonth = normalizeBirthMonth(editEmployee.birthMonth || autoInfo.birthMonth);

    const updatedEmployee: Employee = {
      ...editEmployee,
      name: editEmployee.name!.trim(),
      idNumber: sanitizedIdNumber,
      department: editEmployee.department!.trim(),
      employeeId: editEmployee.employeeId?.trim() || editEmployee.id || "",
      position: editEmployee.position?.trim() || "",
      gender: editEmployee.gender?.trim() || "",
      hireDate: normalizedHireDate,
      age: normalizedAge,
      workYears: normalizedWorkYears,
      birthMonth: normalizedBirthMonth,
      education: editEmployee.education?.trim() || "",
      politicalStatus: editEmployee.politicalStatus?.trim() || "",
      workClothingSize: editEmployee.workClothingSize?.trim() || "",
      safetyShoeSize: editEmployee.safetyShoeSize?.trim() || "",
      householdType: editEmployee.householdType?.trim() || "",
      ethnicity: editEmployee.ethnicity?.trim() || "",
      nativePlace: editEmployee.nativePlace?.trim() || "",
      idAddress: editEmployee.idAddress?.trim() || "",
      maritalStatus: editEmployee.maritalStatus?.trim() || "",
      socialInsurance: editEmployee.socialInsurance?.trim() || "",
      hasBirth: editEmployee.hasBirth?.trim() || "",
      phone: editEmployee.phone?.trim() || "",
      emergencyContact: editEmployee.emergencyContact?.trim() || "",
      emergencyPhone: editEmployee.emergencyPhone?.trim() || "",
      currentAddress: editEmployee.currentAddress?.trim() || "",
      graduateSchool: editEmployee.graduateSchool?.trim() || "",
      major: editEmployee.major?.trim() || "",
      graduationTime: normalizeDateInput(editEmployee.graduationTime),
      email: editEmployee.email?.trim() || "",
      remarks: editEmployee.remarks?.trim() || "",
    } as Employee;

    // 根据员工状态更新对应的列表
    if (updatedEmployee.status === 'active') {
      setEmployees(prev => prev.map(emp => emp.id === updatedEmployee.id ? updatedEmployee : emp));
    } else if (updatedEmployee.status === 'resigned') {
      setResignedEmployees(prev => prev.map(emp => emp.id === updatedEmployee.id ? updatedEmployee : emp));
    }

    setShowEditEmployee(false);
    setEditEmployee({});
    toast.success("员工信息更新成功");
  };

  return (
    <div className={`min-h-screen bg-white ${className || ""}`}>
      <div className="mx-auto flex w-full max-w-7xl flex-col gap-6 p-6 pb-16">
        {/* 页面标题 */}
        <header className="flex flex-col gap-2">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <h1 className="text-3xl font-bold tracking-tight">员工花名册</h1>
              <p className="text-muted-foreground">
                管理员工信息、离职记录和社保增减变动
              </p>
            </div>
          </div>
        </header>

        {/* 主要内容 */}
        <Tabs defaultValue="active" className="w-full">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="active">在职员工</TabsTrigger>
            <TabsTrigger value="resigned">离职员工</TabsTrigger>
            <TabsTrigger value="insurance">社保增减</TabsTrigger>
          </TabsList>

          {/* 在职员工标签页 */}
          <TabsContent value="active" className="space-y-6">
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle>在职员工管理</CardTitle>
                    <CardDescription>
                      管理在职员工信息，支持单个添加和批量导入
                    </CardDescription>
                  </div>
                  <div className="flex gap-2">
                    <Dialog open={showAddEmployee} onOpenChange={setShowAddEmployee}>
                      <DialogTrigger asChild>
                        <Button>
                          <Plus className="h-4 w-4 mr-2" />
                          新增员工
                        </Button>
                      </DialogTrigger>
                      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
                        <DialogHeader>
                          <DialogTitle>新增员工</DialogTitle>
                          <DialogDescription>
                            填写员工详细信息，按分类录入
                          </DialogDescription>
                        </DialogHeader>
                        <Tabs defaultValue="basic" className="w-full">
                          <TabsList className="grid w-full grid-cols-4">
                            <TabsTrigger value="basic">基本信息</TabsTrigger>
                            <TabsTrigger value="personal">个人信息</TabsTrigger>
                            <TabsTrigger value="contact">联系信息</TabsTrigger>
                            <TabsTrigger value="other">其他信息</TabsTrigger>
                          </TabsList>

                          {/* 基本信息 */}
                          <TabsContent value="basic" className="space-y-4">
                            <div className="rounded-lg border bg-muted/20 p-4">
                              <div className="grid gap-4 sm:grid-cols-2">
                              <div className="space-y-2">
                                <Label htmlFor="employeeId">工号</Label>
                                <div className="flex gap-2">
                                  <Input
                                    id="employeeId"
                                    value={newEmployee.employeeId}
                                    onChange={(e) => setNewEmployee(prev => ({ ...prev, employeeId: e.target.value }))}
                                    placeholder="选择部门后可自动生成"
                                  />
                                  <Button
                                    type="button"
                                    variant="outline"
                                    size="sm"
                                    onClick={handleGenerateEmployeeId}
                                    disabled={!newEmployee.department}
                                    title="根据部门规则生成工号"
                                  >
                                    <Dice6 className="h-4 w-4" />
                                  </Button>
                                </div>
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="name">姓名 *</Label>
                                <Input
                                  id="name"
                                  value={newEmployee.name}
                                  onChange={(e) => setNewEmployee(prev => ({ ...prev, name: e.target.value }))}
                                  required
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="department">部门 *</Label>
                                <SearchableSelect
                                  id="department"
                                  value={newEmployee.department || ""}
                                  onChange={(value) => setNewEmployee(prev => ({ ...prev, department: value }))}
                                  options={getDepartments()}
                                  placeholder="选择或搜索部门"
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="position">岗位</Label>
                                <SearchableSelect
                                  id="position"
                                  value={newEmployee.position || ""}
                                  onChange={(value) => setNewEmployee(prev => ({ ...prev, position: value }))}
                                  options={positionOptions}
                                  placeholder="选择或输入岗位"
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="gender">性别</Label>
                                <Select value={newEmployee.gender} onValueChange={(value) => setNewEmployee(prev => ({ ...prev, gender: value }))}>
                                  <SelectTrigger>
                                    <SelectValue placeholder="选择性别" />
                                  </SelectTrigger>
                                  <SelectContent>
                                    <SelectItem value="男">男</SelectItem>
                                    <SelectItem value="女">女</SelectItem>
                                  </SelectContent>
                                </Select>
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="hireDate">入职时间</Label>
                                <Input
                                  id="hireDate"
                                  type="date"
                                  value={newEmployee.hireDate}
                                  onChange={(e) => handleHireDateChange(e.target.value)}
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="age">年龄</Label>
                                <Input
                                  id="age"
                                  type="number"
                                  value={newEmployee.age}
                                  onChange={(e) => setNewEmployee(prev => ({ ...prev, age: e.target.value }))}
                                  onBlur={(e) => handleAgeBlur(e.target.value)}
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="workYears">工龄</Label>
                                <Input
                                  id="workYears"
                                  inputMode="decimal"
                                  value={newEmployee.workYears}
                                  onChange={(e) => setNewEmployee(prev => ({ ...prev, workYears: e.target.value }))}
                                  onBlur={(e) => handleWorkYearsBlur(e.target.value)}
                                />
                              </div>
                              </div>
                            </div>
                          </TabsContent>

                          {/* 个人信息 */}
                          <TabsContent value="personal" className="space-y-4">
                            <div className="rounded-lg border bg-muted/20 p-4 space-y-4">
                            <div className="grid grid-cols-2 gap-4">
                              <div className="space-y-2">
                                <Label htmlFor="birthMonth">出生月份</Label>
                                <Input
                                  id="birthMonth"
                                  value={newEmployee.birthMonth}
                                  onChange={(e) => setNewEmployee(prev => ({ ...prev, birthMonth: e.target.value }))}
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="education">文化程度</Label>
                                <SearchableSelect
                                  id="education"
                                  value={newEmployee.education || ""}
                                  onChange={(value) => setNewEmployee(prev => ({ ...prev, education: value }))}
                                  options={educationOptions}
                                  placeholder="选择或输入文化程度"
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="politicalStatus">政治面貌</Label>
                                <Select value={newEmployee.politicalStatus} onValueChange={(value) => setNewEmployee(prev => ({ ...prev, politicalStatus: value }))}>
                                  <SelectTrigger>
                                    <SelectValue placeholder="选择政治面貌" />
                                  </SelectTrigger>
                                  <SelectContent>
                                    <SelectItem value="群众">群众</SelectItem>
                                    <SelectItem value="团员">团员</SelectItem>
                                    <SelectItem value="党员">党员</SelectItem>
                                    <SelectItem value="民主党派">民主党派</SelectItem>
                                  </SelectContent>
                                </Select>
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="ethnicity">民族</Label>
                                <Input
                                  id="ethnicity"
                                  value={newEmployee.ethnicity}
                                  onChange={(e) => setNewEmployee(prev => ({ ...prev, ethnicity: e.target.value }))}
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="nativePlace">籍贯</Label>
                                <SearchableSelect
                                  id="nativePlace"
                                  value={newEmployee.nativePlace || ""}
                                  onChange={(value) => setNewEmployee(prev => ({ ...prev, nativePlace: value }))}
                                  options={nativePlaceOptions}
                                  placeholder="选择或输入籍贯"
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="maritalStatus">婚姻状况</Label>
                                <Select value={newEmployee.maritalStatus} onValueChange={(value) => setNewEmployee(prev => ({ ...prev, maritalStatus: value }))}>
                                  <SelectTrigger>
                                    <SelectValue placeholder="选择婚姻状况" />
                                  </SelectTrigger>
                                  <SelectContent>
                                    <SelectItem value="未婚">未婚</SelectItem>
                                    <SelectItem value="已婚">已婚</SelectItem>
                                    <SelectItem value="离异">离异</SelectItem>
                                    <SelectItem value="丧偶">丧偶</SelectItem>
                                  </SelectContent>
                                </Select>
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="householdType">户口性质</Label>
                                <Select value={newEmployee.householdType} onValueChange={(value) => setNewEmployee(prev => ({ ...prev, householdType: value }))}>
                                  <SelectTrigger>
                                    <SelectValue placeholder="选择户口性质" />
                                  </SelectTrigger>
                                  <SelectContent>
                                    <SelectItem value="城镇">城镇</SelectItem>
                                    <SelectItem value="农村">农村</SelectItem>
                                  </SelectContent>
                                </Select>
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="hasBirth">是否生育</Label>
                                <Select value={newEmployee.hasBirth} onValueChange={(value) => setNewEmployee(prev => ({ ...prev, hasBirth: value }))}>
                                  <SelectTrigger>
                                    <SelectValue placeholder="选择是否生育" />
                                  </SelectTrigger>
                                  <SelectContent>
                                    <SelectItem value="是">是</SelectItem>
                                    <SelectItem value="否">否</SelectItem>
                                  </SelectContent>
                                </Select>
                              </div>
                            </div>
                            <div className="grid grid-cols-1 gap-4">
                              <div className="space-y-2">
                                <Label htmlFor="idAddress">身份证地址</Label>
                                <Input
                                  id="idAddress"
                                  value={newEmployee.idAddress}
                                  onChange={(e) => setNewEmployee(prev => ({ ...prev, idAddress: e.target.value }))}
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="idNumber">身份证号码 *</Label>
                                <Input
                                  id="idNumber"
                                  value={newEmployee.idNumber}
                                  onChange={(e) => handleIdNumberChange(e.target.value)}
                                  required
                                  placeholder="自动计算年龄和出生月份"
                                />
                              </div>
                            </div>
                            </div>
                          </TabsContent>

                          {/* 联系信息 */}
                          <TabsContent value="contact" className="space-y-4">
                            <div className="rounded-lg border bg-muted/20 p-4 space-y-4">
                            <div className="grid grid-cols-2 gap-4">
                              <div className="space-y-2">
                                <Label htmlFor="phone">联系电话</Label>
                                <Input
                                  id="phone"
                                  value={newEmployee.phone}
                                  onChange={(e) => setNewEmployee(prev => ({ ...prev, phone: e.target.value }))}
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="emergencyContact">紧急联系人</Label>
                                <Input
                                  id="emergencyContact"
                                  value={newEmployee.emergencyContact}
                                  onChange={(e) => setNewEmployee(prev => ({ ...prev, emergencyContact: e.target.value }))}
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="emergencyPhone">紧急联系电话</Label>
                                <Input
                                  id="emergencyPhone"
                                  value={newEmployee.emergencyPhone}
                                  onChange={(e) => setNewEmployee(prev => ({ ...prev, emergencyPhone: e.target.value }))}
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="email">邮箱</Label>
                                <Input
                                  id="email"
                                  type="email"
                                  value={newEmployee.email}
                                  onChange={(e) => setNewEmployee(prev => ({ ...prev, email: e.target.value }))}
                                />
                              </div>
                            </div>
                            <div className="space-y-2">
                              <Label htmlFor="currentAddress">现居住地址</Label>
                              <Input
                                id="currentAddress"
                                value={newEmployee.currentAddress}
                                onChange={(e) => setNewEmployee(prev => ({ ...prev, currentAddress: e.target.value }))}
                              />
                            </div>
                            </div>
                          </TabsContent>

                          {/* 其他信息 */}
                          <TabsContent value="other" className="space-y-4">
                            <div className="rounded-lg border bg-muted/20 p-4 space-y-4">
                            <div className="grid grid-cols-2 gap-4">
                              <div className="space-y-2">
                                <Label htmlFor="workClothingSize">工作服</Label>
                                <Select value={newEmployee.workClothingSize} onValueChange={(value) => setNewEmployee(prev => ({ ...prev, workClothingSize: value }))}>
                                  <SelectTrigger>
                                    <SelectValue placeholder="选择工作服尺码" />
                                  </SelectTrigger>
                                  <SelectContent>
                                    <SelectItem value="S">S</SelectItem>
                                    <SelectItem value="M">M</SelectItem>
                                    <SelectItem value="L">L</SelectItem>
                                    <SelectItem value="XL">XL</SelectItem>
                                    <SelectItem value="XXL">XXL</SelectItem>
                                  </SelectContent>
                                </Select>
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="safetyShoeSize">劳保鞋</Label>
                                <Select value={newEmployee.safetyShoeSize} onValueChange={(value) => setNewEmployee(prev => ({ ...prev, safetyShoeSize: value }))}>
                                  <SelectTrigger>
                                    <SelectValue placeholder="选择劳保鞋尺码" />
                                  </SelectTrigger>
                                  <SelectContent>
                                    {Array.from({length: 15}, (_, i) => 35 + i).map(size => (
                                      <SelectItem key={size} value={size.toString()}>{size}</SelectItem>
                                    ))}
                                  </SelectContent>
                                </Select>
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="socialInsurance">社保</Label>
                                <Select value={newEmployee.socialInsurance} onValueChange={(value) => setNewEmployee(prev => ({ ...prev, socialInsurance: value }))}>
                                  <SelectTrigger>
                                    <SelectValue placeholder="选择社保状态" />
                                  </SelectTrigger>
                                  <SelectContent>
                                    <SelectItem value="有">有</SelectItem>
                                    <SelectItem value="无">无</SelectItem>
                                  </SelectContent>
                                </Select>
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="graduateSchool">毕业院校</Label>
                                <SearchableSelect
                                  id="graduateSchool"
                                  value={newEmployee.graduateSchool || ""}
                                  onChange={(value) => setNewEmployee(prev => ({ ...prev, graduateSchool: value }))}
                                  options={graduateSchoolOptions}
                                  placeholder="选择或输入毕业院校"
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="major">专业</Label>
                                <SearchableSelect
                                  id="major"
                                  value={newEmployee.major || ""}
                                  onChange={(value) => setNewEmployee(prev => ({ ...prev, major: value }))}
                                  options={majorOptions}
                                  placeholder="选择或输入专业"
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="graduationTime">毕业时间</Label>
                                <Input
                                  id="graduationTime"
                                  value={newEmployee.graduationTime}
                                  onChange={(e) => setNewEmployee(prev => ({ ...prev, graduationTime: e.target.value }))}
                                />
                              </div>
                            </div>
                              <div className="space-y-2">
                                <Label htmlFor="remarks">备注</Label>
                                <Input
                                  id="remarks"
                                  value={newEmployee.remarks}
                                  onChange={(e) => setNewEmployee(prev => ({ ...prev, remarks: e.target.value }))}
                                />
                              </div>
                            </div>
                          </TabsContent>
                        </Tabs>
                        <DialogFooter>
                          <Button variant="outline" onClick={() => setShowAddEmployee(false)}>
                            取消
                          </Button>
                          <Button onClick={handleAddEmployee}>添加员工</Button>
                        </DialogFooter>
                      </DialogContent>
                    </Dialog>

                    <Dialog open={showBatchImport} onOpenChange={setShowBatchImport}>
                      <DialogTrigger asChild>
                        <Button variant="outline">
                          <Upload className="h-4 w-4 mr-2" />
                          批量导入
                        </Button>
                      </DialogTrigger>
                      <DialogContent>
                        <DialogHeader>
                          <DialogTitle>批量导入员工</DialogTitle>
                          <DialogDescription>
                            上传Excel文件批量导入员工信息
                          </DialogDescription>
                        </DialogHeader>
                        <div className="grid gap-4 py-4">
                          <div className="space-y-2">
                            <Label>选择文件</Label>
                            <Input
                              ref={fileInputRef}
                              type="file"
                              accept=".xlsx,.xls"
                            />
                            <p className="text-sm text-muted-foreground">
                              支持 .xlsx 和 .xls 格式文件
                            </p>
                          </div>
                          <Button
                            variant="outline"
                            onClick={downloadTemplate}
                            className="w-full"
                          >
                            <Download className="h-4 w-4 mr-2" />
                            下载导入模板
                          </Button>
                        </div>
                        <DialogFooter>
                          <Button variant="outline" onClick={() => setShowBatchImport(false)}>
                            取消
                          </Button>
                          <Button onClick={handleBatchImport} disabled={isLoading}>
                            {isLoading ? "导入中..." : "开始导入"}
                          </Button>
                        </DialogFooter>
                      </DialogContent>
                    </Dialog>

                  </div>
                </div>
              </CardHeader>
              <CardContent>
                {/* 搜索和筛选 */}
                <div className="flex gap-4 mb-4">
                  <div className="flex-1">
                    <div className="relative">
                      <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                      <Input
                        placeholder="搜索员工姓名、身份证号或部门..."
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                        className="pl-10"
                      />
                    </div>
                  </div>
                  <Select value={departmentFilter} onValueChange={setDepartmentFilter}>
                    <SelectTrigger className="w-40">
                      <SelectValue placeholder="选择部门" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">所有部门</SelectItem>
                      {getDepartments().map((dept) => (
                        <SelectItem key={dept} value={dept}>
                          {dept}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <Dialog open={showFieldSelector} onOpenChange={setShowFieldSelector}>
                    <DialogTrigger asChild>
                      <Button variant="outline" size="sm">
                        <Settings className="h-4 w-4 mr-2" />
                        显示字段
                      </Button>
                    </DialogTrigger>
                    <DialogContent className="max-w-md">
                      <DialogHeader>
                        <DialogTitle>自定义显示字段</DialogTitle>
                        <DialogDescription>
                          选择要在表格中显示的字段
                        </DialogDescription>
                      </DialogHeader>
                      <div className="space-y-4">
                        <div className="max-h-80 overflow-y-auto space-y-2">
                          {AVAILABLE_FIELDS.map((field) => (
                            <div key={field.key} className="flex items-center space-x-2">
                              <input
                                type="checkbox"
                                id={`field-${field.key}`}
                                checked={visibleFields.includes(field.key)}
                                onChange={() => handleFieldToggle(field.key)}
                                className="rounded border-gray-300"
                              />
                              <label
                                htmlFor={`field-${field.key}`}
                                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                              >
                                {field.label}
                              </label>
                            </div>
                          ))}
                        </div>
                      </div>
                      <DialogFooter>
                        <Button variant="outline" onClick={resetFieldsToDefault}>
                          恢复默认
                        </Button>
                        <Button onClick={() => setShowFieldSelector(false)}>
                          确定
                        </Button>
                      </DialogFooter>
                    </DialogContent>
                  </Dialog>
                </div>

                {/* 员工列表 */}
                <div className="rounded-md border overflow-x-auto">
                  <Table className="min-w-full">
                    <TableHeader>
                      <TableRow>
                        {getVisibleFieldsConfig().map((field) => (
                          <TableHead
                            key={field.key}
                            style={{ minWidth: field.width }}
                          >
                            {field.label}
                          </TableHead>
                        ))}
                        <TableHead className="min-w-[80px]">操作</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {filteredEmployees.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={visibleFields.length + 1} className="text-center py-8 text-muted-foreground">
                            暂无员工数据
                          </TableCell>
                        </TableRow>
                      ) : (
                        filteredEmployees.map((employee) => (
                          <TableRow
                            key={employee.id}
                            onDoubleClick={() => handleEditEmployee(employee)}
                            className="cursor-pointer hover:bg-muted/50"
                          >
                            {getVisibleFieldsConfig().map((field) => {
                              const displayValue = formatFieldDisplayValue(employee, field.key);
                              return (
                                <TableCell
                                  key={field.key}
                                  className={field.key === 'name' ? 'font-medium' : field.key === 'idNumber' ? 'font-mono text-sm' : 'text-sm'}
                                >
                                  {displayValue}
                                </TableCell>
                              );
                            })}
                            <TableCell>
                              <DropdownMenu>
                                <DropdownMenuTrigger asChild>
                                  <Button variant="ghost" size="sm">
                                    <MoreVertical className="h-4 w-4" />
                                  </Button>
                                </DropdownMenuTrigger>
                                <DropdownMenuContent>
                                  <DropdownMenuItem
                                    onClick={() => {
                                      setSelectedEmployee(employee);
                                      setSingleSocialInsuranceChange({
                                        type: "increase",
                                        effectiveDate: "",
                                        reason: "",
                                      });
                                      setShowSingleSocialInsuranceDialog(true);
                                    }}
                                  >
                                    <CalendarPlus className="h-4 w-4 mr-2" />
                                    社保增加
                                  </DropdownMenuItem>
                                  <DropdownMenuItem
                                    onClick={() => {
                                      setSelectedEmployee(employee);
                                      setSingleSocialInsuranceChange({
                                        type: "decrease",
                                        effectiveDate: "",
                                        reason: "",
                                      });
                                      setShowSingleSocialInsuranceDialog(true);
                                    }}
                                  >
                                    <CalendarMinus className="h-4 w-4 mr-2" />
                                    社保减少
                                  </DropdownMenuItem>
                                  <DropdownMenuItem
                                    onClick={() => {
                                      setSelectedEmployee(employee);
                                      setShowResignDialog(true);
                                    }}
                                  >
                                    <UserMinus className="h-4 w-4 mr-2" />
                                    离职
                                  </DropdownMenuItem>
                                </DropdownMenuContent>
                              </DropdownMenu>
                            </TableCell>
                          </TableRow>
                        ))
                      )}
                    </TableBody>
                  </Table>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          {/* 离职员工标签页 */}
          <TabsContent value="resigned" className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>离职员工</CardTitle>
                <CardDescription>
                  查看已离职员工信息和离职日期
                </CardDescription>
              </CardHeader>
              <CardContent>
                {/* 搜索和筛选 */}
                <div className="flex gap-4 mb-4">
                  <div className="flex-1">
                    <div className="relative">
                      <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                      <Input
                        placeholder="搜索员工姓名、身份证号或部门..."
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                        className="pl-10"
                      />
                    </div>
                  </div>
                  <Select value={departmentFilter} onValueChange={setDepartmentFilter}>
                    <SelectTrigger className="w-40">
                      <SelectValue placeholder="选择部门" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">所有部门</SelectItem>
                      {getDepartments().map((dept) => (
                        <SelectItem key={dept} value={dept}>
                          {dept}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <Dialog open={showFieldSelector} onOpenChange={setShowFieldSelector}>
                    <DialogTrigger asChild>
                      <Button variant="outline" size="sm">
                        <Settings className="h-4 w-4 mr-2" />
                        显示字段
                      </Button>
                    </DialogTrigger>
                    <DialogContent className="max-w-md">
                      <DialogHeader>
                        <DialogTitle>自定义显示字段</DialogTitle>
                        <DialogDescription>
                          选择要在表格中显示的字段
                        </DialogDescription>
                      </DialogHeader>
                      <div className="space-y-4">
                        <div className="max-h-80 overflow-y-auto space-y-2">
                          {AVAILABLE_FIELDS.map((field) => (
                            <div key={field.key} className="flex items-center space-x-2">
                              <input
                                type="checkbox"
                                id={`resigned-field-${field.key}`}
                                checked={visibleFields.includes(field.key)}
                                onChange={() => handleFieldToggle(field.key)}
                                className="rounded border-gray-300"
                              />
                              <label
                                htmlFor={`resigned-field-${field.key}`}
                                className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                              >
                                {field.label}
                              </label>
                            </div>
                          ))}
                        </div>
                      </div>
                      <DialogFooter>
                        <Button variant="outline" onClick={resetFieldsToDefault}>
                          恢复默认
                        </Button>
                        <Button onClick={() => setShowFieldSelector(false)}>
                          确定
                        </Button>
                      </DialogFooter>
                    </DialogContent>
                  </Dialog>
                </div>

                {/* 离职员工列表 */}
                <div className="rounded-md border overflow-x-auto">
                  <Table className="min-w-full">
                    <TableHeader>
                      <TableRow>
                        {getVisibleFieldsConfig().map((field) => (
                          <TableHead
                            key={field.key}
                            style={{ minWidth: field.width }}
                          >
                            {field.label}
                          </TableHead>
                        ))}
                        <TableHead className="min-w-[100px]">离职日期</TableHead>
                        <TableHead className="min-w-[80px]">状态</TableHead>
                        <TableHead className="min-w-[140px]">操作</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {filteredResignedEmployees.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={visibleFields.length + 3} className="text-center py-8 text-muted-foreground">
                            暂无离职员工数据
                          </TableCell>
                        </TableRow>
                      ) : (
                        filteredResignedEmployees.map((employee) => (
                          <TableRow
                            key={employee.id}
                            onDoubleClick={() => handleEditEmployee(employee)}
                            className="cursor-pointer hover:bg-muted/50"
                          >
                            {getVisibleFieldsConfig().map((field) => {
                              const displayValue = formatFieldDisplayValue(employee, field.key);
                              return (
                                <TableCell
                                  key={field.key}
                                  className={field.key === 'name' ? 'font-medium' : field.key === 'idNumber' ? 'font-mono text-sm' : 'text-sm'}
                                >
                                  {displayValue}
                                </TableCell>
                              );
                            })}
                            <TableCell className="text-sm">{normalizeDateInput(employee.resignDate) || "-"}</TableCell>
                            <TableCell>
                              <Badge variant="secondary">已离职</Badge>
                            </TableCell>
                            <TableCell className="flex gap-2">
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={(event) => {
                                  event.stopPropagation();
                                  handleRestoreEmployee(employee);
                                }}
                              >
                                恢复
                              </Button>
                              <Button
                                variant="destructive"
                                size="sm"
                                onClick={(event) => {
                                  event.stopPropagation();
                                  handleRemoveResignedEmployee(employee);
                                }}
                              >
                                删除
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))
                      )}
                    </TableBody>
                  </Table>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          {/* 社保增减标签页 */}
          <TabsContent value="insurance" className="space-y-6">
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle>社保增减管理</CardTitle>
                    <CardDescription>
                      记录员工社保增减变动情况
                    </CardDescription>
                  </div>
                  <Dialog open={showSocialInsuranceDialog} onOpenChange={setShowSocialInsuranceDialog}>
                    <DialogTrigger asChild>
                      <Button>
                        <CalendarPlus className="h-4 w-4 mr-2" />
                        新增变动
                      </Button>
                    </DialogTrigger>
                    <DialogContent>
                      <DialogHeader>
                        <DialogTitle>新增社保变动</DialogTitle>
                        <DialogDescription>
                          选择员工并设置社保增减信息
                        </DialogDescription>
                      </DialogHeader>
                      <div className="grid gap-4 py-4">
                        <div className="grid grid-cols-4 items-center gap-4">
                          <Label className="text-right">变动类型</Label>
                          <Select
                            value={socialInsuranceChange.type}
                            onValueChange={(value: "increase" | "decrease") =>
                              setSocialInsuranceChange(prev => ({ ...prev, type: value }))
                            }
                          >
                            <SelectTrigger className="col-span-3">
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="increase">增加社保</SelectItem>
                              <SelectItem value="decrease">减少社保</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                        <div className="grid grid-cols-4 items-center gap-4">
                          <Label className="text-right">生效日期</Label>
                          <Input
                            type="date"
                            value={socialInsuranceChange.effectiveDate}
                            onChange={(e) => setSocialInsuranceChange(prev => ({ ...prev, effectiveDate: e.target.value }))}
                            className="col-span-3"
                          />
                        </div>
                        <div className="grid grid-cols-4 items-center gap-4">
                          <Label className="text-right">变动原因</Label>
                          <Input
                            value={socialInsuranceChange.reason}
                            onChange={(e) => setSocialInsuranceChange(prev => ({ ...prev, reason: e.target.value }))}
                            placeholder="输入变动原因"
                            className="col-span-3"
                          />
                        </div>
                        <div className="space-y-2">
                          <Label>选择员工</Label>
                          <div className="max-h-40 overflow-y-auto border rounded p-2">
                            {employees.map((employee) => (
                              <div key={employee.id} className="flex items-center space-x-2 py-1">
                                <input
                                  type="checkbox"
                                  id={`emp-${employee.id}`}
                                  checked={socialInsuranceChange.selectedEmployees.includes(employee.id)}
                                  onChange={(e) => {
                                    if (e.target.checked) {
                                      setSocialInsuranceChange(prev => ({
                                        ...prev,
                                        selectedEmployees: [...prev.selectedEmployees, employee.id]
                                      }));
                                    } else {
                                      setSocialInsuranceChange(prev => ({
                                        ...prev,
                                        selectedEmployees: prev.selectedEmployees.filter(id => id !== employee.id)
                                      }));
                                    }
                                  }}
                                />
                                <label htmlFor={`emp-${employee.id}`} className="text-sm">
                                  {employee.name} - {employee.department}
                                </label>
                              </div>
                            ))}
                          </div>
                        </div>
                      </div>
                      <DialogFooter>
                        <Button variant="outline" onClick={() => setShowSocialInsuranceDialog(false)}>
                          取消
                        </Button>
                        <Button onClick={handleSocialInsuranceChange}>确认添加</Button>
                      </DialogFooter>
                    </DialogContent>
                  </Dialog>
                </div>
              </CardHeader>
              <CardContent>
                <div className="rounded-md border">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>员工姓名</TableHead>
                        <TableHead>部门</TableHead>
                        <TableHead>变动类型</TableHead>
                        <TableHead>生效日期</TableHead>
                        <TableHead>变动原因</TableHead>
                        <TableHead>记录日期</TableHead>
                        <TableHead className="min-w-[100px]">操作</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {socialInsuranceChanges.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                            暂无社保变动记录
                          </TableCell>
                        </TableRow>
                      ) : (
                        socialInsuranceChanges.map((change) => (
                          <TableRow key={change.id}>
                            <TableCell className="font-medium">{change.employeeName}</TableCell>
                            <TableCell>{change.department}</TableCell>
                            <TableCell>
                              <Badge variant={change.type === 'increase' ? 'default' : 'destructive'}>
                                {change.type === 'increase' ? '增加' : '减少'}
                              </Badge>
                            </TableCell>
                            <TableCell>{normalizeDateInput(change.effectiveDate) || "-"}</TableCell>
                            <TableCell>{change.reason || "-"}</TableCell>
                            <TableCell>{normalizeDateInput(change.createDate) || "-"}</TableCell>
                            <TableCell>
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => handleRemoveSocialInsuranceChange(change.id)}
                              >
                                撤销
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))
                      )}
                    </TableBody>
                  </Table>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>

        {/* 离职确认对话框 */}
        <Dialog open={showResignDialog} onOpenChange={setShowResignDialog}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>员工离职</DialogTitle>
              <DialogDescription>
                确认将员工 {selectedEmployee?.name} 设置为离职状态
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor="resignDate" className="text-right">
                  离职日期
                </Label>
                <Input
                  id="resignDate"
                  type="date"
                  value={resignDate}
                  onChange={(e) => setResignDate(e.target.value)}
                  className="col-span-3"
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setShowResignDialog(false)}>
                取消
              </Button>
              <Button onClick={handleResignEmployee}>确认离职</Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* 单个员工社保增减对话框 */}
        <Dialog open={showSingleSocialInsuranceDialog} onOpenChange={setShowSingleSocialInsuranceDialog}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>
                {singleSocialInsuranceChange.type === "increase" ? "社保增加" : "社保减少"}
              </DialogTitle>
              <DialogDescription>
                为员工 {selectedEmployee?.name} 添加社保
                {singleSocialInsuranceChange.type === "increase" ? "增加" : "减少"}记录
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor="effectiveDate" className="text-right">
                  生效日期
                </Label>
                <Input
                  id="effectiveDate"
                  type="date"
                  value={singleSocialInsuranceChange.effectiveDate}
                  onChange={(e) =>
                    setSingleSocialInsuranceChange({
                      ...singleSocialInsuranceChange,
                      effectiveDate: e.target.value,
                    })
                  }
                  className="col-span-3"
                />
              </div>
              <div className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor="reason" className="text-right">
                  备注原因
                </Label>
                <Input
                  id="reason"
                  placeholder="可选，填写变更原因"
                  value={singleSocialInsuranceChange.reason}
                  onChange={(e) =>
                    setSingleSocialInsuranceChange({
                      ...singleSocialInsuranceChange,
                      reason: e.target.value,
                    })
                  }
                  className="col-span-3"
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setShowSingleSocialInsuranceDialog(false)}>
                取消
              </Button>
              <Button onClick={handleSingleSocialInsuranceChange}>
                确认{singleSocialInsuranceChange.type === "increase" ? "增加" : "减少"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* 员工编辑对话框 */}
        <Dialog open={showEditEmployee} onOpenChange={setShowEditEmployee}>
          <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>编辑员工信息</DialogTitle>
              <DialogDescription>
                修改员工详细信息，按分类编辑（双击表格行可快速编辑）
              </DialogDescription>
            </DialogHeader>
            <Tabs defaultValue="basic" className="w-full">
              <TabsList className="grid w-full grid-cols-4">
                <TabsTrigger value="basic">基本信息</TabsTrigger>
                <TabsTrigger value="personal">个人信息</TabsTrigger>
                <TabsTrigger value="contact">联系信息</TabsTrigger>
                <TabsTrigger value="other">其他信息</TabsTrigger>
              </TabsList>

              {/* 基本信息标签页 */}
              <TabsContent value="basic" className="space-y-4">
                <div className="rounded-lg border bg-muted/20 p-4">
                  <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="edit-employeeId">工号</Label>
                    <Input
                      id="edit-employeeId"
                      value={editEmployee.employeeId || ""}
                      onChange={(e) => setEditEmployee(prev => ({ ...prev, employeeId: e.target.value }))}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-name">姓名 *</Label>
                    <Input
                      id="edit-name"
                      value={editEmployee.name || ""}
                      onChange={(e) => setEditEmployee(prev => ({ ...prev, name: e.target.value }))}
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-department">部门 *</Label>
                    <SearchableSelect
                      id="edit-department"
                      value={editEmployee.department || ""}
                      onChange={(value) => setEditEmployee(prev => ({ ...prev, department: value }))}
                      options={getDepartments()}
                      placeholder="选择或搜索部门"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-position">岗位</Label>
                    <SearchableSelect
                      id="edit-position"
                      value={editEmployee.position || ""}
                      onChange={(value) => setEditEmployee(prev => ({ ...prev, position: value }))}
                      options={positionOptions}
                      placeholder="选择或输入岗位"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-gender">性别</Label>
                    <Select value={editEmployee.gender || ""} onValueChange={(value) => setEditEmployee(prev => ({ ...prev, gender: value }))}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择性别" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="男">男</SelectItem>
                        <SelectItem value="女">女</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-hireDate">入职时间</Label>
                    <Input
                      id="edit-hireDate"
                      type="date"
                      value={editEmployee.hireDate || ""}
                      onChange={(e) => handleHireDateChange(e.target.value, true)}
                    />
                  </div>
                  </div>
                </div>
              </TabsContent>

              {/* 个人信息标签页 */}
              <TabsContent value="personal" className="space-y-4">
                <div className="rounded-lg border bg-muted/20 p-4 space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="edit-age">年龄</Label>
                    <Input
                      id="edit-age"
                      value={editEmployee.age || ""}
                      onChange={(e) => setEditEmployee(prev => ({ ...prev, age: e.target.value }))}
                      onBlur={(e) => handleAgeBlur(e.target.value, true)}
                      placeholder="可根据身份证自动计算"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-workYears">工龄</Label>
                    <Input
                      id="edit-workYears"
                      inputMode="decimal"
                      value={editEmployee.workYears || ""}
                      onChange={(e) => setEditEmployee(prev => ({ ...prev, workYears: e.target.value }))}
                      onBlur={(e) => handleWorkYearsBlur(e.target.value, true)}
                      placeholder="可根据入职时间自动计算"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-birthMonth">出生月份</Label>
                    <Select value={editEmployee.birthMonth || ""} onValueChange={(value) => setEditEmployee(prev => ({ ...prev, birthMonth: value }))}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择出生月份" />
                      </SelectTrigger>
                      <SelectContent>
                        {Array.from({length: 12}, (_, i) => i + 1).map(month => (
                          <SelectItem key={month} value={`${month}月`}>{month}月</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-education">文化程度</Label>
                    <SearchableSelect
                      id="edit-education"
                      value={editEmployee.education || ""}
                      onChange={(value) => setEditEmployee(prev => ({ ...prev, education: value }))}
                      options={educationOptions}
                      placeholder="选择或输入文化程度"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-politicalStatus">政治面貌</Label>
                    <Select value={editEmployee.politicalStatus || ""} onValueChange={(value) => setEditEmployee(prev => ({ ...prev, politicalStatus: value }))}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择政治面貌" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="群众">群众</SelectItem>
                        <SelectItem value="团员">团员</SelectItem>
                        <SelectItem value="党员">党员</SelectItem>
                        <SelectItem value="民主党派">民主党派</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-ethnicity">民族</Label>
                    <Select value={editEmployee.ethnicity || ""} onValueChange={(value) => setEditEmployee(prev => ({ ...prev, ethnicity: value }))}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择民族" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="汉族">汉族</SelectItem>
                        <SelectItem value="蒙古族">蒙古族</SelectItem>
                        <SelectItem value="回族">回族</SelectItem>
                        <SelectItem value="藏族">藏族</SelectItem>
                        <SelectItem value="维吾尔族">维吾尔族</SelectItem>
                        <SelectItem value="苗族">苗族</SelectItem>
                        <SelectItem value="彝族">彝族</SelectItem>
                        <SelectItem value="壮族">壮族</SelectItem>
                        <SelectItem value="布依族">布依族</SelectItem>
                        <SelectItem value="朝鲜族">朝鲜族</SelectItem>
                        <SelectItem value="其他">其他</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-nativePlace">籍贯</Label>
                    <SearchableSelect
                      id="edit-nativePlace"
                      value={editEmployee.nativePlace || ""}
                      onChange={(value) => setEditEmployee(prev => ({ ...prev, nativePlace: value }))}
                      options={nativePlaceOptions}
                      placeholder="选择或输入籍贯"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-idNumber">身份证号码 *</Label>
                    <Input
                      id="edit-idNumber"
                      value={editEmployee.idNumber || ""}
                      onChange={(e) => handleIdNumberChange(e.target.value, true)}
                      required
                      placeholder="自动计算年龄和出生月份"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-maritalStatus">婚姻状况</Label>
                    <Select value={editEmployee.maritalStatus || ""} onValueChange={(value) => setEditEmployee(prev => ({ ...prev, maritalStatus: value }))}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择婚姻状况" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="未婚">未婚</SelectItem>
                        <SelectItem value="已婚">已婚</SelectItem>
                        <SelectItem value="离异">离异</SelectItem>
                        <SelectItem value="丧偶">丧偶</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-householdType">户口性质</Label>
                    <Select value={editEmployee.householdType || ""} onValueChange={(value) => setEditEmployee(prev => ({ ...prev, householdType: value }))}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择户口性质" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="城镇">城镇</SelectItem>
                        <SelectItem value="农村">农村</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-idAddress">身份证地址</Label>
                    <Input
                      id="edit-idAddress"
                      value={editEmployee.idAddress || ""}
                      onChange={(e) => setEditEmployee(prev => ({ ...prev, idAddress: e.target.value }))}
                    />
                  </div>
                  </div>
                  <div className="grid grid-cols-1 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="edit-idAddress">身份证地址</Label>
                      <Input
                        id="edit-idAddress"
                        value={editEmployee.idAddress || ""}
                        onChange={(e) => setEditEmployee(prev => ({ ...prev, idAddress: e.target.value }))}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-idNumber">身份证号码 *</Label>
                      <Input
                        id="edit-idNumber"
                        value={editEmployee.idNumber || ""}
                        onChange={(e) => handleIdNumberChange(e.target.value, true)}
                        placeholder="自动计算年龄和出生月份"
                      />
                    </div>
                  </div>
                </div>
              </TabsContent>

              {/* 联系信息标签页 */}
              <TabsContent value="contact" className="space-y-4">
                <div className="rounded-lg border bg-muted/20 p-4 space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="edit-phone">联系电话</Label>
                    <Input
                      id="edit-phone"
                      value={editEmployee.phone || ""}
                      onChange={(e) => setEditEmployee(prev => ({ ...prev, phone: e.target.value }))}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-email">邮箱</Label>
                    <Input
                      id="edit-email"
                      type="email"
                      value={editEmployee.email || ""}
                      onChange={(e) => setEditEmployee(prev => ({ ...prev, email: e.target.value }))}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-emergencyContact">紧急联系人</Label>
                    <Input
                      id="edit-emergencyContact"
                      value={editEmployee.emergencyContact || ""}
                      onChange={(e) => setEditEmployee(prev => ({ ...prev, emergencyContact: e.target.value }))}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-emergencyPhone">家庭电话/紧急情况联系电话</Label>
                    <Input
                      id="edit-emergencyPhone"
                      value={editEmployee.emergencyPhone || ""}
                      onChange={(e) => setEditEmployee(prev => ({ ...prev, emergencyPhone: e.target.value }))}
                    />
                  </div>
                  <div className="col-span-2 space-y-2">
                    <Label htmlFor="edit-currentAddress">现居住地址</Label>
                    <Input
                      id="edit-currentAddress"
                      value={editEmployee.currentAddress || ""}
                      onChange={(e) => setEditEmployee(prev => ({ ...prev, currentAddress: e.target.value }))}
                    />
                  </div>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-currentAddress">现居住地址</Label>
                    <Input
                      id="edit-currentAddress"
                      value={editEmployee.currentAddress || ""}
                      onChange={(e) => setEditEmployee(prev => ({ ...prev, currentAddress: e.target.value }))}
                    />
                  </div>
                </div>
              </TabsContent>

              {/* 其他信息标签页 */}
              <TabsContent value="other" className="space-y-4">
                <div className="rounded-lg border bg-muted/20 p-4 space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="edit-workClothingSize">工作服</Label>
                    <Select value={editEmployee.workClothingSize || ""} onValueChange={(value) => setEditEmployee(prev => ({ ...prev, workClothingSize: value }))}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择工作服尺码" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="XS">XS</SelectItem>
                        <SelectItem value="S">S</SelectItem>
                        <SelectItem value="M">M</SelectItem>
                        <SelectItem value="L">L</SelectItem>
                        <SelectItem value="XL">XL</SelectItem>
                        <SelectItem value="XXL">XXL</SelectItem>
                        <SelectItem value="XXXL">XXXL</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-safetyShoeSize">劳保鞋</Label>
                    <Select value={editEmployee.safetyShoeSize || ""} onValueChange={(value) => setEditEmployee(prev => ({ ...prev, safetyShoeSize: value }))}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择劳保鞋尺码" />
                      </SelectTrigger>
                      <SelectContent>
                        {Array.from({length: 20}, (_, i) => i + 35).map(size => (
                          <SelectItem key={size} value={size.toString()}>{size}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-socialInsurance">社保</Label>
                    <Select value={editEmployee.socialInsurance || ""} onValueChange={(value) => setEditEmployee(prev => ({ ...prev, socialInsurance: value }))}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择社保状态" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="有">有</SelectItem>
                        <SelectItem value="无">无</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-hasBirth">是否生育</Label>
                    <Select value={editEmployee.hasBirth || ""} onValueChange={(value) => setEditEmployee(prev => ({ ...prev, hasBirth: value }))}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择是否生育" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="是">是</SelectItem>
                        <SelectItem value="否">否</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-graduateSchool">毕业院校</Label>
                    <SearchableSelect
                      id="edit-graduateSchool"
                      value={editEmployee.graduateSchool || ""}
                      onChange={(value) => setEditEmployee(prev => ({ ...prev, graduateSchool: value }))}
                      options={graduateSchoolOptions}
                      placeholder="选择或输入毕业院校"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-major">专业</Label>
                    <SearchableSelect
                      id="edit-major"
                      value={editEmployee.major || ""}
                      onChange={(value) => setEditEmployee(prev => ({ ...prev, major: value }))}
                      options={majorOptions}
                      placeholder="选择或输入专业"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-graduationTime">毕业时间</Label>
                    <Input
                      id="edit-graduationTime"
                      value={editEmployee.graduationTime || ""}
                      onChange={(e) => setEditEmployee(prev => ({ ...prev, graduationTime: e.target.value }))}
                      placeholder="例如：2019"
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="edit-remarks">备注</Label>
                  <Input
                    id="edit-remarks"
                    value={editEmployee.remarks || ""}
                    onChange={(e) => setEditEmployee(prev => ({ ...prev, remarks: e.target.value }))}
                  />
                  </div>
                </div>
              </TabsContent>
            </Tabs>
            <DialogFooter>
              <Button variant="outline" onClick={() => setShowEditEmployee(false)}>
                取消
              </Button>
              <Button onClick={handleSaveEmployee}>保存修改</Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  );
}

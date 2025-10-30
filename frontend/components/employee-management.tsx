"use client";

import { useState, useRef } from "react";
import { toast } from "sonner";
import {
  Users,
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
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

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

  // 部门列表（简化版，用于选择）
  const departments = [
    "总经办", "财务部", "销售部", "仓库", "生产部",
    "机电部", "技术质量部", "人事行政部"
  ];


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
  const [newEmployee, setNewEmployee] = useState<Partial<Employee>>({
    employeeId: "",
    name: "",
    department: "",
    position: "",
    gender: "",
    hireDate: "",
    age: "",
    workYears: "",
    birthMonth: "",
    education: "初中",
    politicalStatus: "群众",
    workClothingSize: "L",
    safetyShoeSize: "40",
    householdType: "",
    ethnicity: "",
    nativePlace: "",
    idAddress: "",
    idNumber: "",
    maritalStatus: "",
    socialInsurance: "有",
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
    employees.forEach(emp => emp.department && depts.add(emp.department));
    resignedEmployees.forEach(emp => emp.department && depts.add(emp.department));
    return Array.from(depts).sort();
  };

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
      emp.department.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesDepartment = !departmentFilter || departmentFilter === "all" || emp.department === departmentFilter;

    return matchesSearch && matchesDepartment;
  });

  // 筛选离职员工
  const filteredResignedEmployees = resignedEmployees.filter(emp => {
    const matchesSearch = !searchTerm ||
      emp.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      emp.idNumber.includes(searchTerm) ||
      emp.department.toLowerCase().includes(searchTerm.toLowerCase());

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
    const existingEmployee = employees.find(emp => emp.idNumber === newEmployee.idNumber);
    if (existingEmployee) {
      toast.error("该身份证号已存在");
      return;
    }

    const employee: Employee = {
      id: Date.now().toString(),
      employeeId: newEmployee.employeeId || Date.now().toString(),
      name: newEmployee.name.trim(),
      department: newEmployee.department,
      position: newEmployee.position || "",
      gender: newEmployee.gender || "",
      hireDate: newEmployee.hireDate || new Date().toISOString().split('T')[0],
      age: newEmployee.age || "",
      workYears: newEmployee.workYears || "",
      birthMonth: newEmployee.birthMonth || "",
      education: newEmployee.education || "",
      politicalStatus: newEmployee.politicalStatus || "",
      workClothingSize: newEmployee.workClothingSize || "",
      safetyShoeSize: newEmployee.safetyShoeSize || "",
      householdType: newEmployee.householdType || "",
      ethnicity: newEmployee.ethnicity || "",
      nativePlace: newEmployee.nativePlace || "",
      idAddress: newEmployee.idAddress || "",
      idNumber: newEmployee.idNumber.trim(),
      maritalStatus: newEmployee.maritalStatus || "",
      socialInsurance: newEmployee.socialInsurance || "",
      hasBirth: newEmployee.hasBirth || "",
      phone: newEmployee.phone || "",
      emergencyContact: newEmployee.emergencyContact || "",
      emergencyPhone: newEmployee.emergencyPhone || "",
      currentAddress: newEmployee.currentAddress || "",
      graduateSchool: newEmployee.graduateSchool || "",
      major: newEmployee.major || "",
      graduationTime: newEmployee.graduationTime || "",
      email: newEmployee.email || "",
      remarks: newEmployee.remarks || "",
      status: 'active',
    };

    setEmployees(prev => [...prev, employee]);
    setNewEmployee({
      employeeId: "",
      name: "",
      department: "",
      position: "",
      gender: "",
      hireDate: "",
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
    setShowAddEmployee(false);
    toast.success("员工添加成功");
  };

  // 批量导入员工
  const handleBatchImport = () => {
    const file = fileInputRef.current?.files?.[0];
    if (!file) {
      toast.error("请选择文件");
      return;
    }

    setIsLoading(true);
    // 这里应该调用API处理Excel文件
    setTimeout(() => {
      toast.success("批量导入成功");
      setIsLoading(false);
      setShowBatchImport(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    }, 2000);
  };

  // 员工离职
  const handleResignEmployee = () => {
    if (!selectedEmployee || !resignDate) {
      toast.error("请选择离职日期");
      return;
    }

    const resignedEmployee: Employee = {
      ...selectedEmployee,
      status: 'resigned',
      resignDate,
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
        department: employee?.department || "",
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
      department: selectedEmployee.department,
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

  // 下载模板
  const downloadTemplate = () => {
    // 这里应该调用API下载模板
    toast.info("正在下载模板...");
  };

  // 从身份证号码计算年龄和出生月份
  const calculateAgeFromIdNumber = (idNumber: string) => {
    if (!idNumber || idNumber.length !== 18) return { age: "", birthMonth: "" };

    const birth = idNumber.substring(6, 14);
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
      birthMonth: `${birthMonth}月`
    };
  };

  // 从入职时间计算工龄
  const calculateWorkYears = (hireDate: string) => {
    if (!hireDate) return "";

    const hire = new Date(hireDate);
    const today = new Date();
    const diffTime = Math.abs(today.getTime() - hire.getTime());
    const diffYears = diffTime / (1000 * 60 * 60 * 24 * 365.25);

    return Math.floor(diffYears).toString();
  };

  // 生成工号
  const generateEmployeeId = (department: string) => {
    const departmentCode = DEPARTMENT_CODES[department as keyof typeof DEPARTMENT_CODES];
    if (!departmentCode) return "";

    // 查找该部门最大的工号
    const departmentEmployees = [...employees, ...resignedEmployees]
      .filter(emp => emp.employeeId && emp.employeeId.startsWith(departmentCode))
      .map(emp => emp.employeeId)
      .sort();

    if (departmentEmployees.length === 0) {
      return `${departmentCode}001`;
    }

    const lastId = departmentEmployees[departmentEmployees.length - 1];
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
    const { age, birthMonth } = calculateAgeFromIdNumber(idNumber);

    if (isEdit) {
      setEditEmployee(prev => ({
        ...prev,
        idNumber,
        age: age || prev.age,
        birthMonth: birthMonth || prev.birthMonth
      }));
    } else {
      setNewEmployee(prev => ({
        ...prev,
        idNumber,
        age: age || prev.age,
        birthMonth: birthMonth || prev.birthMonth
      }));
    }
  };

  // 处理入职时间变化
  const handleHireDateChange = (hireDate: string, isEdit = false) => {
    const workYears = calculateWorkYears(hireDate);

    if (isEdit) {
      setEditEmployee(prev => ({
        ...prev,
        hireDate,
        workYears: workYears || prev.workYears
      }));
    } else {
      setNewEmployee(prev => ({
        ...prev,
        hireDate,
        workYears: workYears || prev.workYears
      }));
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
    const existingEmployee = [...employees, ...resignedEmployees].find(
      emp => emp.idNumber === editEmployee.idNumber && emp.id !== editEmployee.id
    );
    if (existingEmployee) {
      toast.error("该身份证号已存在");
      return;
    }

    const updatedEmployee: Employee = {
      ...editEmployee,
      name: editEmployee.name!.trim(),
      idNumber: editEmployee.idNumber!.trim(),
      department: editEmployee.department!,
      employeeId: editEmployee.employeeId || editEmployee.id || "",
      position: editEmployee.position || "",
      gender: editEmployee.gender || "",
      hireDate: editEmployee.hireDate || "",
      age: editEmployee.age || "",
      workYears: editEmployee.workYears || "",
      birthMonth: editEmployee.birthMonth || "",
      education: editEmployee.education || "",
      politicalStatus: editEmployee.politicalStatus || "",
      workClothingSize: editEmployee.workClothingSize || "",
      safetyShoeSize: editEmployee.safetyShoeSize || "",
      householdType: editEmployee.householdType || "",
      ethnicity: editEmployee.ethnicity || "",
      nativePlace: editEmployee.nativePlace || "",
      idAddress: editEmployee.idAddress || "",
      maritalStatus: editEmployee.maritalStatus || "",
      socialInsurance: editEmployee.socialInsurance || "",
      hasBirth: editEmployee.hasBirth || "",
      phone: editEmployee.phone || "",
      emergencyContact: editEmployee.emergencyContact || "",
      emergencyPhone: editEmployee.emergencyPhone || "",
      currentAddress: editEmployee.currentAddress || "",
      graduateSchool: editEmployee.graduateSchool || "",
      major: editEmployee.major || "",
      graduationTime: editEmployee.graduationTime || "",
      email: editEmployee.email || "",
      remarks: editEmployee.remarks || "",
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
                            <div className="grid grid-cols-2 gap-4">
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
                                <Select value={newEmployee.department || ""} onValueChange={(value) => setNewEmployee(prev => ({ ...prev, department: value }))}>
                                  <SelectTrigger>
                                    <SelectValue placeholder="选择部门" />
                                  </SelectTrigger>
                                  <SelectContent>
                                    {getDepartments().map((dept) => (
                                      <SelectItem key={dept} value={dept}>{dept}</SelectItem>
                                    ))}
                                    <SelectItem value="总经办">总经办</SelectItem>
                                    <SelectItem value="销售部">销售部</SelectItem>
                                    <SelectItem value="生产部">生产部</SelectItem>
                                    <SelectItem value="机电部">机电部</SelectItem>
                                    <SelectItem value="财务部">财务部</SelectItem>
                                    <SelectItem value="人事行政部">人事行政部</SelectItem>
                                    <SelectItem value="技术质量部">技术质量部</SelectItem>
                                    <SelectItem value="仓库">仓库</SelectItem>
                                  </SelectContent>
                                </Select>
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="position">岗位</Label>
                                <Input
                                  id="position"
                                  value={newEmployee.position}
                                  onChange={(e) => setNewEmployee(prev => ({ ...prev, position: e.target.value }))}
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
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="workYears">工龄</Label>
                                <Input
                                  id="workYears"
                                  value={newEmployee.workYears}
                                  onChange={(e) => setNewEmployee(prev => ({ ...prev, workYears: e.target.value }))}
                                />
                              </div>
                            </div>
                          </TabsContent>

                          {/* 个人信息 */}
                          <TabsContent value="personal" className="space-y-4">
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
                                <Select value={newEmployee.education} onValueChange={(value) => setNewEmployee(prev => ({ ...prev, education: value }))}>
                                  <SelectTrigger>
                                    <SelectValue placeholder="选择文化程度" />
                                  </SelectTrigger>
                                  <SelectContent>
                                    <SelectItem value="小学">小学</SelectItem>
                                    <SelectItem value="初中">初中</SelectItem>
                                    <SelectItem value="高中">高中</SelectItem>
                                    <SelectItem value="中专">中专</SelectItem>
                                    <SelectItem value="大专">大专</SelectItem>
                                    <SelectItem value="本科">本科</SelectItem>
                                    <SelectItem value="硕士">硕士</SelectItem>
                                    <SelectItem value="博士">博士</SelectItem>
                                  </SelectContent>
                                </Select>
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
                                <Input
                                  id="nativePlace"
                                  value={newEmployee.nativePlace}
                                  onChange={(e) => setNewEmployee(prev => ({ ...prev, nativePlace: e.target.value }))}
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
                          </TabsContent>

                          {/* 联系信息 */}
                          <TabsContent value="contact" className="space-y-4">
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
                          </TabsContent>

                          {/* 其他信息 */}
                          <TabsContent value="other" className="space-y-4">
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
                                <Input
                                  id="graduateSchool"
                                  value={newEmployee.graduateSchool}
                                  onChange={(e) => setNewEmployee(prev => ({ ...prev, graduateSchool: e.target.value }))}
                                />
                              </div>
                              <div className="space-y-2">
                                <Label htmlFor="major">专业</Label>
                                <Input
                                  id="major"
                                  value={newEmployee.major}
                                  onChange={(e) => setNewEmployee(prev => ({ ...prev, major: e.target.value }))}
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
                              const value = employee[field.key];
                              return (
                                <TableCell
                                  key={field.key}
                                  className={field.key === 'name' ? 'font-medium' : field.key === 'idNumber' ? 'font-mono text-sm' : 'text-sm'}
                                >
                                  {value || "-"}
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
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {filteredResignedEmployees.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={visibleFields.length + 2} className="text-center py-8 text-muted-foreground">
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
                              const value = employee[field.key];
                              return (
                                <TableCell
                                  key={field.key}
                                  className={field.key === 'name' ? 'font-medium' : field.key === 'idNumber' ? 'font-mono text-sm' : 'text-sm'}
                                >
                                  {value || "-"}
                                </TableCell>
                              );
                            })}
                            <TableCell className="text-sm">{employee.resignDate}</TableCell>
                            <TableCell>
                              <Badge variant="secondary">已离职</Badge>
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
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {socialInsuranceChanges.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
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
                            <TableCell>{change.effectiveDate}</TableCell>
                            <TableCell>{change.reason}</TableCell>
                            <TableCell>{change.createDate}</TableCell>
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
                    <Select value={editEmployee.department || ""} onValueChange={(value) => setEditEmployee(prev => ({ ...prev, department: value }))}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择部门" />
                      </SelectTrigger>
                      <SelectContent>
                        {getDepartments().map((dept) => (
                          <SelectItem key={dept} value={dept}>{dept}</SelectItem>
                        ))}
                        <SelectItem value="销售部">销售部</SelectItem>
                        <SelectItem value="技术部">技术部</SelectItem>
                        <SelectItem value="财务部">财务部</SelectItem>
                        <SelectItem value="人事部">人事部</SelectItem>
                        <SelectItem value="行政部">行政部</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-position">岗位</Label>
                    <Input
                      id="edit-position"
                      value={editEmployee.position || ""}
                      onChange={(e) => setEditEmployee(prev => ({ ...prev, position: e.target.value }))}
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
              </TabsContent>

              {/* 个人信息标签页 */}
              <TabsContent value="personal" className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="edit-age">年龄</Label>
                    <Input
                      id="edit-age"
                      value={editEmployee.age || ""}
                      onChange={(e) => setEditEmployee(prev => ({ ...prev, age: e.target.value }))}
                      placeholder="可根据身份证自动计算"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-workYears">工龄</Label>
                    <Input
                      id="edit-workYears"
                      value={editEmployee.workYears || ""}
                      onChange={(e) => setEditEmployee(prev => ({ ...prev, workYears: e.target.value }))}
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
                    <Select value={editEmployee.education || ""} onValueChange={(value) => setEditEmployee(prev => ({ ...prev, education: value }))}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择文化程度" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="小学">小学</SelectItem>
                        <SelectItem value="初中">初中</SelectItem>
                        <SelectItem value="高中">高中</SelectItem>
                        <SelectItem value="中专">中专</SelectItem>
                        <SelectItem value="大专">大专</SelectItem>
                        <SelectItem value="本科">本科</SelectItem>
                        <SelectItem value="硕士">硕士</SelectItem>
                        <SelectItem value="博士">博士</SelectItem>
                      </SelectContent>
                    </Select>
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
                    <Input
                      id="edit-nativePlace"
                      value={editEmployee.nativePlace || ""}
                      onChange={(e) => setEditEmployee(prev => ({ ...prev, nativePlace: e.target.value }))}
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
              </TabsContent>

              {/* 联系信息标签页 */}
              <TabsContent value="contact" className="space-y-4">
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
              </TabsContent>

              {/* 其他信息标签页 */}
              <TabsContent value="other" className="space-y-4">
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
                    <Input
                      id="edit-graduateSchool"
                      value={editEmployee.graduateSchool || ""}
                      onChange={(e) => setEditEmployee(prev => ({ ...prev, graduateSchool: e.target.value }))}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="edit-major">专业</Label>
                    <Input
                      id="edit-major"
                      value={editEmployee.major || ""}
                      onChange={(e) => setEditEmployee(prev => ({ ...prev, major: e.target.value }))}
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
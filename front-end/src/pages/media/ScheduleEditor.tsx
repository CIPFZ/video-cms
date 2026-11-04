import React, { useEffect, useState } from 'react';
import { Alert, Form, Input, Modal, Select, Switch, message } from 'antd';
import { t, useI18n } from '@/i18n';
import { updateMediaSchedule, type Schedule } from '@/services/media';
import { modeNames } from './shared';
const presets = () => [
 {value:'*/20 * * * *',label:'每 20 分钟'},
 {value:'15 */3 * * *',label:'每 3 小时 · 第 15 分钟'},
 {value:'0 3 * * *',label:'每天 · 03:00'},
 {value:'30 3 * * 0',label:'每周日 · 03:30'},
 {value:'30 3 1 * *',label:t('cms.media.schedule.monthly')},
 {value:'custom',label:'自定义 Cron'},
];
export default function ScheduleEditor({ schedule, sourceName, fullAllowed, onClose, onSaved }: { schedule?: Schedule; sourceName?: string; fullAllowed?: boolean; onClose:()=>void; onSaved:()=>void }) {
 useI18n();
 const [form]=Form.useForm();const [saving,setSaving]=useState(false),[preset,setPreset]=useState('custom');
 useEffect(()=>{if(schedule){form.setFieldsValue({cron:schedule.cron,enabled:schedule.enabled});setPreset(presets().some(p=>p.value===schedule.cron)?schedule.cron:'custom')}},[schedule]);
 const locked=schedule?.kind==='full'&&!fullAllowed;
 const save=async()=>{const values=await form.validateFields();if(!schedule)return;setSaving(true);try{const r=await updateMediaSchedule(schedule.id,values);if(r.code!==0)throw Error(r.msg);message.success('执行计划已保存');onSaved();onClose()}catch(e){message.error(e instanceof Error?e.message:'保存失败')}finally{setSaving(false)}};
 return <Modal open={!!schedule} title={sourceName+' · '+modeNames[schedule?.kind||'']} onCancel={onClose} onOk={save} confirmLoading={saving} okText="保存计划" destroyOnHidden>
 <Form form={form} layout="vertical" style={{marginTop:24}}>
 {locked&&<Alert type="warning" showIcon message="此站点未开放全量采集" description="请先在资源站点编辑中开启“允许全量采集”。" style={{marginBottom:20}}/>}
 <Form.Item label="执行周期"><Select value={preset} options={presets()} onChange={v=>{setPreset(v);if(v!=='custom')form.setFieldValue('cron',v)}}/></Form.Item>
 <Form.Item name="cron" label="Cron 表达式" extra="分 时 日 月 周 · 时区 Asia/Shanghai（北京时间）" rules={[{required:true,message:'请输入 Cron'},{validator:(_,v)=>v?.trim().split(/\s+/).length===5?Promise.resolve():Promise.reject(new Error('请输入五段 Cron 表达式'))}]}><Input readOnly={preset!=='custom'} placeholder="0 3 * * *"/></Form.Item>
 <Form.Item name="enabled" label="启用定时执行" valuePropName="checked"><Switch disabled={locked}/></Form.Item>
 <Alert type="info" showIcon message="保存后按新周期计算下次执行时间；停用计划不会中断正在执行的任务。"/>
 </Form></Modal>;
}

import React, { useEffect, useState } from 'react';
import { PageContainer } from '@ant-design/pro-components';
import { Button, Card, Col, Form, Input, InputNumber, Modal, Row, Space, Switch, Table, Tag, Typography, message } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { createMediaSource, listMediaSources, updateMediaSource, type MediaSource, type MediaSourceInput } from '@/services/media';
import { t, useI18n } from '@/i18n';
import '../style.less';

const defaults: MediaSourceInput = { name: '', apiUrl: '', apiType: 'json', enabled: true, priority: 100, fullEnabled: false, allowCategories: '', denyCategories: '', requestIntervalSeconds: 3, timeoutSeconds: 15 };
export default function Sources() {
 useI18n();
 const [rows,setRows]=useState<MediaSource[]>([]), [loading,setLoading]=useState(false), [open,setOpen]=useState(false), [saving,setSaving]=useState(false), [editing,setEditing]=useState<MediaSource>();
 const [query,setQuery]=useState(''); const [form]=Form.useForm<MediaSourceInput>();
 const load=async()=>{setLoading(true);try{const r=await listMediaSources();if(r.code!==0)throw Error(r.msg);setRows(r.data||[])}catch{message.error('读取资源站点失败')}finally{setLoading(false)}};
 useEffect(()=>{void load()},[]);
 const edit=(row?:MediaSource)=>{setEditing(row);form.resetFields();form.setFieldsValue(row||defaults);setOpen(true)};
 const save=async()=>{const values=await form.validateFields();setSaving(true);try{const r=editing?await updateMediaSource(editing.id,values):await createMediaSource({...defaults,...values});if(r.code!==0)throw Error(r.msg);message.success('站点配置已保存');setOpen(false);await load()}catch(e){message.error(e instanceof Error?e.message:'保存失败')}finally{setSaving(false)}};
 return <PageContainer title={false}><div className="media-workspace">
 <Card className="media-panel"><div className="media-toolbar"><Input.Search allowClear placeholder="搜索站点名称或接口" value={query} onChange={e=>setQuery(e.target.value)} style={{maxWidth:340}}/><Space><Typography.Text type="secondary">{rows.length} 个站点</Typography.Text><Button icon={<ReloadOutlined/>} onClick={load}>刷新</Button><Button type="primary" icon={<PlusOutlined/>} onClick={()=>edit()}>添加站点</Button></Space></div>
 <Table<MediaSource> rowKey="id" loading={loading} dataSource={rows.filter(s=>(s.name+s.apiUrl).toLowerCase().includes(query.toLowerCase()))} pagination={{ pageSize: 10, showSizeChanger: false }} scroll={{x:850}} columns={[
 {title:'站点',render:(_,s)=><Space size={14}><span className="media-source-dot">{s.name.slice(0,1)}</span><div><Typography.Text strong>{s.name}</Typography.Text><div className="media-muted">JSON 资源接口</div></div></Space>},
 {title:'接口地址',dataIndex:'apiUrl',render:v=><Typography.Text copyable ellipsis={{tooltip:v}} style={{maxWidth:330}}>{v}</Typography.Text>},
 {title:'状态',render:(_,s)=><Tag color={s.suspended?'orange':s.enabled?'green':'default'}>{s.suspended?'自动暂挂':s.enabled?'启用':'停用'}</Tag>},
 {title:'采集范围',render:(_,s)=><Typography.Text type="secondary">{s.allowCategories||'全部分类'}{s.denyCategories?' · 已设排除项':''}</Typography.Text>},
 {title:'优先级',dataIndex:'priority',width:90},
 {title:'操作',width:90,render:(_,s)=><Button type="link" onClick={()=>edit(s)}>编辑</Button>}
 ]}/></Card>
 <Modal title={editing?'编辑资源站点':'添加资源站点'} open={open} onCancel={()=>setOpen(false)} onOk={save} confirmLoading={saving} okText="保存配置" width={680}>
 <Form form={form} layout="vertical" initialValues={defaults} style={{marginTop:24}}>
 <Row gutter={20}><Col span={16}><Form.Item name="name" label="站点名称" rules={[{required:true,message:'请输入站点名称'}]}><Input maxLength={100}/></Form.Item></Col><Col span={8}><Form.Item name="priority" label="优先级 · 数字越小越优先"><InputNumber min={1} max={10000} style={{width:'100%'}}/></Form.Item></Col></Row>
 <Form.Item name="apiUrl" label="资源 API 地址" rules={[{required:true,message:'请输入接口地址'},{type:'url',message:'请输入完整 URL'}]}><Input placeholder="https://example.com/api.php/provide/vod/"/></Form.Item>
 <Row gutter={20}><Col span={12}><Form.Item name="allowCategories" label="采集分类" extra="留空表示全部；多项使用逗号分隔"><Input placeholder="电影,电视剧,动漫"/></Form.Item></Col><Col span={12}><Form.Item name="denyCategories" label="排除分类" extra="排除项优先于采集分类"><Input placeholder="伦理,短视频,直播"/></Form.Item></Col></Row>
 <Row gutter={20}><Col span={12}><Form.Item name="requestIntervalSeconds" label="请求间隔（秒）"><InputNumber min={0} max={60} style={{width:'100%'}}/></Form.Item></Col><Col span={12}><Form.Item name="timeoutSeconds" label="请求超时（秒）"><InputNumber min={1} max={120} style={{width:'100%'}}/></Form.Item></Col></Row>
 <Row gutter={20}><Col span={12}><Form.Item name="enabled" label="启用站点" valuePropName="checked"><Switch/></Form.Item></Col><Col span={12}><Form.Item name="fullEnabled" label="允许全量采集" valuePropName="checked" extra={t('cms.media.sources.fullEnabledHint')}><Switch/></Form.Item></Col></Row>
 </Form></Modal></div></PageContainer>;
}

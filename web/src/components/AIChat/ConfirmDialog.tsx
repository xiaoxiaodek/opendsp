import { Modal, Descriptions, Typography } from 'antd';

const { Text } = Typography;

interface Props {
  open: boolean;
  tool: string;
  args: Record<string, unknown>;
  onConfirm: () => void;
  onCancel: () => void;
}

const toolLabels: Record<string, string> = {
  update_campaign_budget: 'Update Campaign Budget',
  update_adgroup_bid: 'Update Ad Group Bid',
  update_adgroup_status: 'Toggle Ad Group Status',
  update_campaign_status: 'Toggle Campaign Status',
  audit_creative: 'Audit Creative',
};

export default function ConfirmDialog({ open, tool, args, onConfirm, onCancel }: Props) {
  return (
    <Modal
      title="Confirm Action"
      open={open}
      onOk={onConfirm}
      onCancel={onCancel}
      okText="Confirm"
      cancelText="Cancel"
    >
      <Text>AI wants to perform the following action:</Text>
      <Descriptions column={1} size="small" style={{ marginTop: 12 }}>
        <Descriptions.Item label="Action">
          <Text strong>{toolLabels[tool] || tool}</Text>
        </Descriptions.Item>
        {Object.entries(args).map(([key, value]) => (
          <Descriptions.Item key={key} label={key}>
            {String(value)}
          </Descriptions.Item>
        ))}
      </Descriptions>
    </Modal>
  );
}

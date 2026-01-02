const BASE_URL = '/api';

export interface CurrentStatus {
  profile: string;
  modified: boolean;
  custom: boolean;
}

export interface ProfilesResponse {
  profiles: string[];
}

export async function getProfiles(tool: string): Promise<string[]> {
  const res = await fetch(`${BASE_URL}/${tool}/profiles`);
  if (!res.ok) throw new Error(await res.text());
  const data: ProfilesResponse = await res.json();
  return data.profiles || [];
}

export async function getCurrent(tool: string): Promise<CurrentStatus> {
  const res = await fetch(`${BASE_URL}/${tool}/current`);
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function saveProfile(tool: string, profile: string, force: boolean = false): Promise<void> {
  const res = await fetch(`${BASE_URL}/${tool}/profiles`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ profile, force }),
  });
  if (!res.ok) {
    const data = await res.json();
    throw new Error(data.error || 'Failed to save profile');
  }
}

export async function switchProfile(tool: string, profile: string): Promise<void> {
  const res = await fetch(`${BASE_URL}/${tool}/switch/${encodeURIComponent(profile)}`, {
    method: 'POST',
  });
  if (!res.ok) {
    const data = await res.json();
    throw new Error(data.error || 'Failed to switch profile');
  }
}

export async function deleteProfile(tool: string, profile: string): Promise<boolean> {
  const res = await fetch(`${BASE_URL}/${tool}/profiles/${encodeURIComponent(profile)}`, {
    method: 'DELETE',
  });
  if (!res.ok) {
    const data = await res.json();
    throw new Error(data.error || 'Failed to delete profile');
  }
  const data = await res.json();
  return data.cleared;
}
